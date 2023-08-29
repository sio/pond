package test

import (
	"github.com/sio/pond/lib/block"
	"testing"

	"bytes"
	"embed"
	"os"
	"strings"
)

var fields = map[string]string{
	"SimpleField": "hello\nworld",
	"Some special characters (\u200b 世界)": "Special\nCharacters\nRendered\nCorrectly:\n\u200b 世界",
}

//go:embed sample*
var samples embed.FS

func TestTemplateFill(t *testing.T) {
	files, _ := samples.ReadDir(".")
	for _, file := range files {
		t.Run(file.Name(), func(t *testing.T) {
			raw, _ := samples.ReadFile(file.Name())
			input := bytes.NewReader(raw)
			template := new(block.Template)
			for key, value := range fields {
				template.Set(key, value)
			}
			output, err := template.Fill(input)
			if err != nil {
				t.Fatalf("failed to fill template: %v", err)
			}

			outputPath := strings.Replace(file.Name(), ".in", ".out", 1)
			if saveOutput && !strings.HasSuffix(file.Name(), ".out") {
				err = os.WriteFile(outputPath, output, 0644)
				if err != nil {
					t.Fatalf("failed to save rendered output: %v", err)
				}
				t.Logf("saved output: %s", outputPath)
				return
			}

			expected, _ := samples.ReadFile(outputPath)
			if !bytes.Equal(output, expected) {
				t.Fatalf("template rendered incorrectly:\nwant:\n%s\ngot:\n%s", string(expected), string(output))
			}
		})
	}
}
