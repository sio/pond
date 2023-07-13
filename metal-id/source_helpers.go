package metal_id

import (
	"os"
)

// Checks whether path is pointing to a directory
//
// This function is ignoring best practices on purpose because:
//   - dir.IsDir() does not consider symlinks to directories to be directories
//   - filepath.Join() loses typically meaningless but important in this case
//     "/." at the end of path
func isDir(path string) bool {
	_, err := os.Stat(path + "/.")
	return !os.IsNotExist(err)
}
