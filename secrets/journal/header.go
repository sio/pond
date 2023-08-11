package journal

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/ssh"

	"secrets/util"
)

const (
	applicationTag     = "pond/secrets"
	v1                 = "JOURNAL VERSION 1"
	v1NonceBytes       = 24
	v1KeyBytes         = 32
	v1SeparatorBytes   = 4
	v1HeaderFieldCount = 5
)

var (
	errEmptyStream = errors.New("empty stream")
)

func (j *Journal) parseHeader() error {
	if !j.ready() {
		return errors.New("can not parse uninitialized journal")
	}
	// TODO: If new journal versions are to be supported, version detection
	// TODO: needs to be lifted out of v1ParseHeader() into this function
	return j.v1ParseHeader()
}

func (j *Journal) writeHeader() error {
	return j.v1WriteHeader()
}

func (j *Journal) v1WriteHeader() error {
	if !j.ready() {
		return errors.New("can not generate header for uninitialized journal")
	}
	var h v1Header
	h.Application = applicationTag
	h.Version = v1
	h.PublicKey = util.KeyText(j.signer.PublicKey())
	h.SetTimestamp(time.Now())
	nonce := make([]byte, v1NonceBytes)
	if len(j.state) >= v1NonceBytes {
		nonce = j.state[:v1NonceBytes]
	}
	h.SetNonce(nonce)

	var err error
	err = j.v1InitializeJournal(&h)
	if err != nil {
		return err
	}
	_, err = j.stream.Write(h.Bytes())
	return err
}

func (j *Journal) v1ParseHeader() error {
	var fields [v1HeaderFieldCount]string
	for i := 0; i < len(fields); i++ {
		fragment, err := readBytes(j.stream, byte('\n'))
		if errors.Is(err, io.EOF) && i == 0 && len(fragment) == 0 {
			return errEmptyStream
		}
		if err != nil {
			return fmt.Errorf("reading header: %w", err)
		}
		fields[i] = string(fragment[:len(fragment)-1])
	}
	var h v1Header
	h.FromStrings(fields)
	return j.v1InitializeJournal(&h)
}

// Similar to bufio.Reader.ReadBytes, this function reads until the first
// occurence of delim in input and returns a slice containing the data up to
// and including the delimiter.
//
// Unlike bufio.Reader and bufio.Scanner this function never advances the
// reader past the position of delimiter.
//
// Reading bytes one by one has a significant performance penalty which is
// acceptable only for short data streams. Use bufio.Reader or bufio.Scanner if
// need better performance.
func readBytes(r io.Reader, delim byte) ([]byte, error) {
	const growBytes = 1024
	var output []byte
	for i := 0; true; i++ {
		if i > len(output)-1 {
			output = append(output, make([]byte, growBytes)...)
		}
		_, err := r.Read(output[i : i+1])
		if err != nil {
			return nil, err
		}
		if output[i] == delim {
			return output[:i+1], nil
		}
	}
	panic("impossible branching")
}

func (j *Journal) v1InitializeJournal(h *v1Header) error {
	var err error

	// Magic values
	if h.Application != applicationTag {
		return fmt.Errorf("unexpected application tag: %q", h.Application)
	}
	if h.Version != v1 {
		return fmt.Errorf("unsupported journal version: %q", h.Version)
	}
	j.version = h.Version

	// Public key
	pubkey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(h.PublicKey))
	if err != nil {
		return fmt.Errorf("parsing public key: %w", err)
	}
	var input = make([]byte, 32)
	_, err = io.ReadFull(rand.Reader, input)
	if err != nil {
		return fmt.Errorf("rand: %w", err)
	}
	signature, err := j.signer.Sign(rand.Reader, input)
	if err != nil {
		return fmt.Errorf("sign: %w", err)
	}
	err = pubkey.Verify(input, signature)
	if err != nil {
		return fmt.Errorf("public key verification: %w", err)
	}

	// Timestamp
	ctime, err := h.GetTimestamp()
	if err != nil {
		return fmt.Errorf("timestamp: %w", err)
	}
	j.ctime = ctime

	// Initialize message reader state
	nonce, err := h.GetNonce()
	if err != nil {
		return fmt.Errorf("nonce: %w", err)
	}
	if len(nonce) != v1NonceBytes {
		return fmt.Errorf("invalid nonce length for %s: %d instead of %d", j.version, len(nonce), v1NonceBytes)
	}
	signature, err = j.signer.Sign(rand.Reader, h.Bytes())
	if err != nil {
		return fmt.Errorf("sign: %w", err)
	}
	if len(signature.Blob) < 64 {
		return fmt.Errorf("signature is too short: %d bytes instead of at least 64", len(signature.Blob))
	}
	var derived, key []byte
	salt := sha256.Sum256(signature.Blob)
	derived = argon2.IDKey(signature.Blob, salt[:], 4, 256*1024, 2, v1KeyBytes+v1SeparatorBytes)
	j.separator, key = derived[:v1SeparatorBytes], derived[v1SeparatorBytes:]
	j.state = append(nonce, key...)
	return nil
}

type v1Header struct {
	Application string
	Version     string
	PublicKey   string
	Timestamp   string
	Nonce       string
}

func (h *v1Header) String() string {
	s := h.ToStrings()
	var b strings.Builder
	for _, line := range s {
		b.WriteString(line)
		b.WriteRune('\n')
	}
	return b.String()
}

func (h *v1Header) Bytes() []byte {
	return []byte(h.String())
}

func (h *v1Header) ToStrings() [v1HeaderFieldCount]string {
	var s [v1HeaderFieldCount]string
	s[0] = h.Application
	s[1] = h.Version
	s[2] = h.PublicKey
	s[3] = h.Timestamp
	s[4] = h.Nonce
	return s
}

func (h *v1Header) FromStrings(s [v1HeaderFieldCount]string) {
	h.Application = s[0]
	h.Version = s[1]
	h.PublicKey = s[2]
	h.Timestamp = s[3]
	h.Nonce = s[4]
}

func (h *v1Header) GetNonce() ([]byte, error) {
	return base64.StdEncoding.DecodeString(h.Nonce)
}

func (h *v1Header) SetNonce(nonce []byte) {
	h.Nonce = base64.StdEncoding.EncodeToString(nonce)
}

func (h *v1Header) GetTimestamp() (time.Time, error) {
	u, e := strconv.Atoi(h.Timestamp)
	if e != nil {
		return time.Time{}, fmt.Errorf("parsing unix timestamp: %w", e)
	}
	return time.Unix(int64(u), 0), nil
}

func (h *v1Header) SetTimestamp(t time.Time) {
	h.Timestamp = fmt.Sprint(t.Unix())
}
