// Simple block based file templating engine
package block

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Update content of predefined section blocks in a Markdown document
type Template struct {
	sections map[string]string
}

func (t *Template) Set(key, value string) {
	if t.sections == nil {
		t.sections = make(map[string]string)
	}
	t.sections[key] = strings.TrimRight(value, " \t\n\r")
}

var (
	startMarker = regexp.MustCompile(`--SECTION (.*) START(?: OFFSET (\d+)|)--`)
	endMarker   = regexp.MustCompile(`--SECTION (.*) END(?: OFFSET (\d+)|)--`)
)

func (t *Template) Fill(src io.Reader) ([]byte, error) {
	var (
		scanner       = bufio.NewScanner(src)
		output        = new(bytes.Buffer)
		section, next string
		index, offset int
		buffer, match []string
		err           error
		rendered      = make(map[string]bool)
	)
	for scanner.Scan() {
		// Start marker detected
		if match = startMarker.FindStringSubmatch(scanner.Text()); len(match) != 0 {
			next = match[1]
			offset = 0
			if len(match) > 2 && match[2] != "" {
				offset, err = strconv.Atoi(match[2])
				if err != nil {
					return nil, err
				}
			}
			fmt.Fprintln(output, scanner.Text())
			_, exist := t.sections[next]
			if exist {
				section = next
				index = 0
				buffer = make([]string, offset)
			}
			continue
		}

		// End marker detected
		if match = endMarker.FindStringSubmatch(scanner.Text()); section != "" && len(match) != 0 {
			next = match[1]
			offset = 0
			if len(match) > 2 && match[2] != "" {
				offset, err = strconv.Atoi(match[2])
				if err != nil {
					return nil, err
				}
			}
			if next != section {
				return nil, fmt.Errorf("section tag mismatch:\n open: %s\nclose: %s", section, next)
			}
			fmt.Fprintln(output, t.sections[section])
			rendered[section] = true
			if len(buffer) != 0 {
				for i := 0; i < offset; i++ {
					fmt.Fprintln(output, buffer[(index-offset+i)%len(buffer)])
				}
				buffer = nil
			}
			fmt.Fprintln(output, scanner.Text())
			section = ""
			continue
		}

		// Skip existing section content
		if len(buffer) != 0 {
			buffer[index%len(buffer)] = scanner.Text()
		}
		index++
		if section != "" && index > offset {
			continue
		}

		// Default line processing
		fmt.Fprintln(output, scanner.Text())
	}
	if err = scanner.Err(); err != nil {
		return nil, err
	}
	var missing = make([]string, 0, len(t.sections))
	for section := range t.sections {
		if !rendered[section] {
			missing = append(missing, section)
		}
	}
	if len(missing) != 0 {
		return nil, fmt.Errorf("%d provided blocks not rendered: %s", len(missing), strings.Join(missing, "; "))
	}
	return output.Bytes(), nil
}

func (t *Template) Render(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	content, err := t.Fill(file)
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}
	file, err = os.OpenFile(filepath, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	err = file.Truncate(0)
	if err != nil {
		return err
	}
	_, err = file.Write(content)
	if err != nil {
		return err
	}
	return nil
}
