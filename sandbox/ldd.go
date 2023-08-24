package sandbox

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
)

func ldd(path string) (libs []string, err error) {
	output := new(bytes.Buffer)
	ldd := exec.Command("ldd", path)
	ldd.Stdout = output
	ldd.Stderr = output
	err = ldd.Run()
	if err != nil {
		return nil, err
	}
	libs = make([]string, 0)
	for _, line := range bytes.Split(output.Bytes(), []byte{'\n'}) {
		if len(line) < 2 {
			continue
		}
		match := lddArrowName.FindSubmatch(line)
		if len(match) != 2 {
			match = lddPlainName.FindSubmatch(line)
			if len(match) != 2 {
				return nil, fmt.Errorf("unable to parse line: %s", string(line))
			}
		}
		lib := string(match[1])
		_, err = os.Stat(lib)
		if err != nil {
			continue
		}
		libs = append(libs, lib)
		continue
	}
	return libs, nil
}

var (
	lddPlainName = regexp.MustCompile(`\s*(.*) \(0x`)
	lddArrowName = regexp.MustCompile(`.* => (.*) \(0x`)
)
