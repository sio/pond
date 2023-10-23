package main

import (
	"fmt"
	"os"
	"strings"
	"unicode"
)

func previewSeed(unsafe bool) func(data []byte) string {
	return func(data []byte) string {
		const (
			maxPreviewLength  = 80 - 8 - 10
			nonPrintableByte  = '.'
			safePrintableByte = 'x'
		)
		var builder strings.Builder
		for index, b := range data {
			if index > maxPreviewLength {
				break
			}
			var char = rune(b)
			if !unicode.IsPrint(char) {
				char = nonPrintableByte
			} else if !unsafe {
				char = safePrintableByte
			}
			builder.WriteRune(char)
		}
		builder.WriteString(fmt.Sprintf(" [%d bytes]", len(data)))
		return builder.String()
	}
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
