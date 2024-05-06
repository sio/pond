package s3

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type remoteInterface interface {
	// Partial reader for a limited chunk of remote object
	Reader(ctx context.Context, offset, length int64) (io.ReadCloser, error)

	// Full size of remote object
	Size() int64

	io.Closer
}

func openMinioRemote(endpoint, access, secret, bucket, object string) (remoteInterface, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("empty endpoint URL")
	}
	if bucket == "" {
		return nil, fmt.Errorf("empty bucket name")
	}
	if object == "" {
		return nil, fmt.Errorf("empty object name")
	}
	remote, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	var useTLS bool = true
	switch remote.Scheme {
	case "http":
		useTLS = false
	default:
	}
	m := new(minioRemote)
	m.client, err = minio.New(remote.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(access, secret, ""),
		Secure: useTLS,
	})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*10))
	defer cancel()
	stat, err := m.client.StatObject(ctx, bucket, object, minio.StatObjectOptions{})
	if err != nil {
		return nil, err
	}
	m.size = stat.Size
	m.bucket, m.object = bucket, object
	return m, nil
}

type minioRemote struct {
	client         *minio.Client
	bucket, object string
	size           int64
}

func (m *minioRemote) Size() int64 {
	return m.size
}

func (m *minioRemote) Close() error {
	return nil // minio.Client does not require any cleanup
}

func (m *minioRemote) Reader(ctx context.Context, offset, length int64) (io.ReadCloser, error) {
	if offset > m.size || length <= 0 || offset < 0 {
		return nullReader{}, nil
	}
	end := offset + length - 1
	if end > m.size {
		end = m.size
	}

	get := minio.GetObjectOptions{}
	err := get.SetRange(offset, end)
	if err != nil {
		return nil, fmt.Errorf("set range: %w", err)
	}
	object, err := m.client.GetObject(ctx, m.bucket, m.object, get)
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}
	return object, nil
}

type nullReader struct{}

func (r nullReader) Read(p []byte) (int, error) {
	return 0, io.EOF
}

func (r nullReader) Close() error {
	return nil
}
