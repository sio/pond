package pid1

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/sio/pond/initramfs/kmod"
)

func loadDeviceModules() error {
	seen := make(map[string]struct{})
	return filepath.WalkDir("/sys", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if filepath.Base(path) != "modalias" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil // skip devices we have no access to
		}
		modalias := strings.TrimSpace(string(raw))
		if modalias == "" {
			return nil
		}
		if _, done := seen[modalias]; done {
			return nil
		}
		seen[modalias] = struct{}{}
		return kmod.LoadAlias(modalias)
	})
}
