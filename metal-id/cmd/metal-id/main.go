package main

import (
	"metal_id"

	"crypto"
	"fmt"
	"os"
	"strings"
	"unicode"
)

func main() {
	var hwid = metal_id.New()

	var err error
	for _, data := range metal_id.Sources() {
		stderr("Reading from %s", data.Name)
		for {
			var chunk []byte
			chunk = data.Next()
			if len(chunk) == 0 {
				break
			}
			stderr(previewSeedData(chunk))
			_, err = hwid.Write(chunk)
			if err != nil {
				fail("Failed to add data to fingerprint: %v", err)
			}
		}
	}

	var key crypto.Signer
	key, err = hwid.Key()
	if err != nil {
		fail("Failed to generate machine key: %v", err)
	}

	var output []byte
	output, err = metal_id.EncodePublicKey(key.Public())
	if err != nil {
		fail("Failed to encode public key: %v", err)
	}
	fmt.Println(string(output))
}

func stderr(format string, a ...any) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	_, err := fmt.Fprintf(os.Stderr, format, a...)
	if err != nil {
		panic("failed to write to stderr: " + fmt.Sprint(err))
	}
}

func fail(format string, a ...any) {
	stderr(format, a...)
	os.Exit(1)
}

func previewSeedData(data []byte) string {
	const (
		maxPreviewLength = 80 - 8 - 10
		nonPrintableByte = '.'
	)
	var builder strings.Builder
	for index, b := range data {
		if index > maxPreviewLength {
			break
		}
		var char = rune(b)
		if !unicode.IsPrint(char) {
			char = nonPrintableByte
		}
		builder.WriteRune(char)
	}
	builder.WriteString(fmt.Sprintf(" [%d bytes]", len(data)))
	return builder.String()
}
