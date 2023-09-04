package value

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	fieldColumnWidth = 9 // base on longest key width
	blobDelimiter    = "---"
	blobLineWidth    = 86
)

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
	record(buf, "Signer", string(ssh.MarshalAuthorizedKey(v.Signer)))
	_, err = io.Copy(out, buf)
	if err != nil {
		return err
	}
	err = writeBlob64(out, v.Signature)
	if err != nil {
		return err
	}
	err = writeBlob64(out, v.Blob)
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
	_, err := fmt.Fprintln(out, blobDelimiter)
	if err != nil {
		return err
	}
	for i := 0; i < len(encoded); i = end {
		end = i + blobLineWidth
		if end > len(encoded) {
			end = len(encoded)
		}
		_, err = fmt.Fprintln(out, string(encoded[i:end]))
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
		err         error
		lineNo      uint
		next        Value
		readingBlob bool
		blobBuffer  = new(bytes.Buffer)
		readingSig  bool
		sigBuffer   = new(bytes.Buffer)
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
			readingSig = !readingSig
			if !readingSig {
				readingBlob = !readingBlob
			}
		case readingSig:
			sigBuffer.WriteString(line)
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
			next.Signer, _, _, _, err = ssh.ParseAuthorizedKey([]byte(value))
			if err != nil {
				return fmt.Errorf("line #%d: invalid signer: %v", lineNo, err)
			}
		default:
			return fmt.Errorf("line #%d: invalid field: %s", lineNo, field)
		}
	}
	if scanner.Err() != nil {
		return scanner.Err()
	}
	next.Signature, err = base64.StdEncoding.DecodeString(sigBuffer.String())
	if err != nil {
		return fmt.Errorf("decoding base64 signature: %w", err)
	}
	next.Blob, err = base64.StdEncoding.DecodeString(blobBuffer.String())
	if err != nil {
		return fmt.Errorf("decoding base64 blob: %w", err)
	}
	*v = next
	err = v.Verify()
	if err != nil {
		return err
	}
	return nil
}
