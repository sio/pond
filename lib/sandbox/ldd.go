package sandbox

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// This function works by shelling out to `ldd` utility and parsing its output.
// Not recommended for deployment
func Ldd(path string) (libs []string, err error) {
	return ldd(path)
}

func ldd(path string) (libs []string, err error) {
	output := new(bytes.Buffer)
	ldd := exec.Command("ldd", path)
	ldd.Stdout = output
	ldd.Stderr = output
	ldd.Env = os.Environ()
	ldd.Env = append(ldd.Env, "LANG=C", "LC_ALL=C")
	err = ldd.Run()
	if err != nil {
		if ldd.ProcessState.ExitCode() == 1 &&
			strings.TrimSpace(output.String()) == "not a dynamic executable" {
			return nil, nil // empty output for static binaries
		}
		return nil, fmt.Errorf("%w: %s", err, output.String())
	}
	libs = make([]string, 0)
	for _, line := range bytes.Split(output.Bytes(), []byte{'\n'}) {
		if len(line) < 2 {
			continue
		}
		if strings.TrimSpace(string(line)) == "statically linked" {
			continue
		}
		match := lddArrowName.FindSubmatch(line)
		if len(match) != 2 {
			match = lddPlainName.FindSubmatch(line)
		}
		if len(match) != 2 {
			return nil, fmt.Errorf("unable to parse line: %s", string(line))
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
