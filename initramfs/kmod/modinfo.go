/* Fetch some metadata from a binary Linux kernel module file */
package kmod

import (
	"bytes"
	"debug/elf"
	"fmt"
	"strings"
)

type Modinfo struct {
	Name    string
	Depends []string
	Alias   []string
}

func (mod Modinfo) String() string {
	repr := fmt.Sprintf("%s (depends on %d modules, has %d aliases)", mod.Name, len(mod.Depends), len(mod.Alias))
	if len(mod.Depends) > 0 {
		repr += fmt.Sprintf("\n\tDepends: %s", strings.Join(mod.Depends, ", "))
	}
	if len(mod.Alias) > 0 {
		repr += fmt.Sprintf("\n\tAliases:\n\t\t%s", strings.Join(mod.Alias, "\n\t\t"))
	}
	return repr
}

func Info(path string) (info Modinfo, err error) {
	bin, err := elf.Open(path)
	if err != nil {
		return info, err
	}
	defer func() { _ = bin.Close() }()
	section := bin.Section(".modinfo")
	if section == nil {
		return info, fmt.Errorf("section .modinfo not found in %s", path)
	}
	data, err := section.Data()
	if err != nil {
		return info, fmt.Errorf("failed to read .modinfo from %s: %w", path, err)
	}
	var line, value string
	var found bool
	for _, row := range bytes.Split(data, []byte{0}) {
		line = string(row)
		value, found = strings.CutPrefix(line, "alias=")
		if found && value != "" {
			info.Alias = append(info.Alias, value)
			continue
		}
		value, found = strings.CutPrefix(line, "depends=")
		if found && value != "" {
			info.Depends = append(info.Depends, strings.Split(value, ",")...)
			continue
		}
		value, found = strings.CutPrefix(line, "name=")
		if found && value != "" {
			info.Name = value
			continue
		}
	}
	return info, nil
}
