package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/myparsleycat/ddsutil"
)

func copyTestdata(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readDds(t *testing.T, path string) *ddsutil.Dds {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	dds, err := ddsutil.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	return dds
}

func TestRetagLegacyInPlace(t *testing.T) {
	path := copyTestdata(t, "bc1.dds")
	before := readDds(t, path)
	if err := retag(path, path, "bc1_unorm_srgb", false); err != nil {
		t.Fatal(err)
	}
	after := readDds(t, path)
	if after.Header10 == nil || after.Header10.DxgiFormat != ddsutil.DxgiFormatBC1_UNorm_sRGB {
		t.Fatalf("Header10 = %+v", after.Header10)
	}
	if string(after.Data) != string(before.Data) {
		t.Fatal("retag changed the data")
	}
	if format, err := after.ImageFormat(); err != nil || format != ddsutil.BC1RgbaUnormSrgb {
		t.Fatalf("ImageFormat = %v, %v", format, err)
	}
}

func TestRetagRejectsSizeMismatch(t *testing.T) {
	path := copyTestdata(t, "bc1.dds")
	output := filepath.Join(t.TempDir(), "out.dds")
	if err := retag(path, output, "BC7_UNorm", false); err == nil {
		t.Fatal("retag BC1 to BC7 succeeded without -force")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("rejected retag wrote the output")
	}
	if err := retag(path, output, "BC7_UNorm", true); err != nil {
		t.Fatalf("retag with -force: %v", err)
	}
	if err := retag(path, output, "BC9", true); err == nil {
		t.Fatal("unknown format accepted")
	}
}

func TestRetagForceUpdatesLinearSize(t *testing.T) {
	path := copyTestdata(t, "bc1.dds")
	if err := retag(path, path, "BC7_UNorm", true); err != nil {
		t.Fatal(err)
	}
	dds := readDds(t, path)
	// 8x8 BC7 is 4 blocks of 16 bytes.
	if dds.Header.LinearSize == nil || *dds.Header.LinearSize != 64 || dds.Header.Pitch != nil {
		t.Fatalf("pitch %v, linear size %v; want linear size 64", dds.Header.Pitch, dds.Header.LinearSize)
	}
	if !dds.Header.Flags.Contains(ddsutil.HeaderFlagsLINEARSIZE) || dds.Header.Flags.Contains(ddsutil.HeaderFlagsPITCH) {
		t.Fatalf("flags %v", dds.Header.Flags)
	}
}

func TestRetagThroughSymlink(t *testing.T) {
	target := copyTestdata(t, "bc1.dds")
	link := filepath.Join(t.TempDir(), "link.dds")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	if err := retag(link, link, "BC1_UNorm_sRGB", false); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("link was replaced: %v, %v", info, err)
	}
	if dds := readDds(t, target); dds.Header10 == nil || dds.Header10.DxgiFormat != ddsutil.DxgiFormatBC1_UNorm_sRGB {
		t.Fatal("link target was not retagged")
	}
}
