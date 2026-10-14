//go:build unix

package main

import (
	"os"
	"syscall"
	"testing"
)

func TestRetagKeepsOwner(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("changing the owner of a file requires root")
	}
	path := copyTestdata(t, "bc1.dds")
	const uid, gid = 1234, 5678
	if err := os.Chown(path, uid, gid); err != nil {
		t.Fatal(err)
	}
	if err := retag(path, path, "BC1_UNorm_sRGB", false); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	stat := info.Sys().(*syscall.Stat_t)
	if stat.Uid != uid || stat.Gid != gid {
		t.Fatalf("owner %d:%d, want %d:%d", stat.Uid, stat.Gid, uid, gid)
	}
}
