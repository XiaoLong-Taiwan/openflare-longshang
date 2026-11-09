//go:build !windows

package localfile

import "os"

func syncDirectory(directory *os.File) error {
	return directory.Sync()
}
