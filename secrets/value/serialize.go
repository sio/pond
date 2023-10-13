package value

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/sio/pond/lib/bytepack"
)

const (
	fieldColumnWidth = 7 // base on longest key width
	blobDelimiter    = "---"
	blobLineWidth    = 72
)

func Load(filename string) (*Value, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	v := new(Value)
	err = v.Deserialize(file)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (v *Value) Serialize(out io.Writer) error {
	err := v.Verify()
	if err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}
	var buf = new(bytes.Buffer)
	record(buf, fileHeader, "")
	for _, p := range v.Path {
		record(buf, "Path", p)
	}
	record(buf, "Created", v.Created.UTC().Format(time.RFC3339))
	record(buf, "Expires", v.Expires.UTC().Format(time.RFC3339))
	record(buf, "Signer", fmt.Sprintf("%s (%s)", ssh.FingerprintSHA256(v.Signer), v.Signer.Type()))
	record(buf, blobDelimiter, "")
	_, err = io.Copy(out, buf)
	if err != nil {
		return err
	}
	pack, err := bytepack.Pack([][]byte{
		ssh.MarshalAuthorizedKey(v.Signer),
		v.signature,
		v.blob,
	})
	if err != nil {
		return err
	}
	err = writeBlob64(out, pack.Blob())
	if err != nil {
		return err
	}
	return nil
}

func record(buf *bytes.Buffer, key, value string) {
	if len(value) > 0 {
		value = strings.TrimRight(value, "\n\r")
		_, _ = fmt.Fprintf(buf, "%-*s %s\n", fieldColumnWidth, key, value)
	} else {
		_, _ = fmt.Fprintln(buf, key)
	}
}

func writeBlob64(out io.Writer, data []byte) error {
	var encoded = make([]byte, base64.StdEncoding.EncodedLen(len(data)))
	base64.StdEncoding.Encode(encoded, data)
	var end int
	for i := 0; i < len(encoded); i = end {
		end = i + blobLineWidth
		if end > len(encoded) {
			end = len(encoded)
		}
		_, err := fmt.Fprintln(out, string(encoded[i:end]))
		if err != nil {
			return err
		}
	}
	return nil
}

func (v *Value) Deserialize(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		return errors.New("empty input")
	}
	if scanner.Text() != fileHeader {
		return fmt.Errorf("unexpected file header: %s", scanner.Text())
	}
	var (
		blobBuffer  = new(bytes.Buffer)
		err         error
		fpSigner    string
		lineNo      uint
		next        Value
		readingBlob bool
	)
	for scanner.Scan() {
		line := strings.TrimLeft(scanner.Text(), " \t")
		field, value, ok := strings.Cut(line, " ")
		if ok {
			value = strings.TrimLeft(value, " \t")
		}
		lineNo++
		switch {
		case line == "" || strings.HasPrefix(line, "#"):
			// skip empty lines and comments
		case line == blobDelimiter:
			readingBlob = !readingBlob
		case readingBlob:
			blobBuffer.WriteString(line)
		case !ok:
			return fmt.Errorf("line #%d: failed to parse field name", lineNo)
		case field == "Path":
			next.Path = append(next.Path, value)
		case field == "Created":
			next.Created, err = time.Parse(time.RFC3339, value)
			if err != nil {
				return fmt.Errorf("line #%d: invalid timestamp: %v", lineNo, err)
			}
		case field == "Expires":
			next.Expires, err = time.Parse(time.RFC3339, value)
			if err != nil {
				return fmt.Errorf("line #%d: invalid timestamp: %v", lineNo, err)
			}
		case field == "Signer":
			fpSigner, _, _ = strings.Cut(value, " ")
		default:
			return fmt.Errorf("line #%d: invalid field: %s", lineNo, field)
		}
	}
	if scanner.Err() != nil {
		return scanner.Err()
	}
	blob, err := base64.StdEncoding.DecodeString(blobBuffer.String())
	if err != nil {
		return fmt.Errorf("decoding base64 blob: %w", err)
	}
	pack, err := bytepack.Wrap(blob)
	if err != nil {
		return fmt.Errorf("unpacking blob: %w", err)
	}
	if pack.Size() != 3 {
		return fmt.Errorf("unexpected number of blob elements: %d (instead of 3)", pack.Size())
	}
	next.Signer, _, _, _, err = ssh.ParseAuthorizedKey(pack.Element(0))
	if err != nil {
		return fmt.Errorf("invalid signer public key: %v", err)
	}
	if fpSigner != ssh.FingerprintSHA256(next.Signer) {
		return fmt.Errorf("signer fingerprint (%s) does not match the one used in signature (%s)", fpSigner, ssh.FingerprintSHA256(next.Signer))
	}
	next.signature = pack.Element(1)
	next.blob = pack.Element(2)
	*v = next
	err = v.Verify()
	if err != nil {
		return err
	}
	return nil
}
