//go:build unix

package main

import (
	"os"
	"syscall"
)

// copyOwner gives file the owner and group of info, best effort: changing
// the owner needs privileges, so without them only the group is copied (if
// the user belongs to it), and otherwise the file keeps the user's own.
func copyOwner(file *os.File, info os.FileInfo) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return
	}
	if file.Chown(int(stat.Uid), int(stat.Gid)) != nil {
		_ = file.Chown(-1, int(stat.Gid))
	}
}
