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

// TestRetagComparesEachMip checks layouts whose total sizes match but whose
// mip levels differ: a 1x3 RGBA16F layer with 2 mips is 24+8 bytes, the same
// layer in BC2 is 16+16 bytes.
func TestRetagComparesEachMip(t *testing.T) {
	dds, err := ddsutil.NewDxgi(ddsutil.DxgiParams{Width: 1, Height: 3, Format: ddsutil.DxgiFormatR16G16B16A16_Float, MipmapLevels: 2})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "in.dds")
	data, err := dds.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := retag(path, path, "BC2_UNorm", false); err == nil {
		t.Fatal("retag with a different mip layout succeeded without -force")
	}
}

func TestRetagRejectsVideoFormats(t *testing.T) {
	dds, err := ddsutil.NewDxgi(ddsutil.DxgiParams{Width: 4, Height: 4, Format: ddsutil.DxgiFormatR8G8B8A8_UNorm})
	if err != nil {
		t.Fatal(err)
	}
	dds.Header10.DxgiFormat = ddsutil.DxgiFormatP010
	path := filepath.Join(t.TempDir(), "in.dds")
	data, err := dds.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	// P010 and Y210 both report 10 bits per pixel, but their layouts differ.
	if err := retag(path, path, "Y210", false); err == nil {
		t.Fatal("retag between video formats succeeded without -force")
	}
	if err := retag(path, path, "Y210", true); err != nil {
		t.Fatalf("retag with -force: %v", err)
	}
}

func TestRetagKeepsLegacyPremultipliedAlpha(t *testing.T) {
	for _, tc := range []struct {
		format ddsutil.D3DFormat
		name   string
		want   ddsutil.AlphaMode
	}{
		{ddsutil.D3DFormatDXT2, "bc2_unorm", ddsutil.AlphaModePreMultiplied},
		{ddsutil.D3DFormatDXT3, "bc2_unorm", ddsutil.AlphaModeUnknown},
		{ddsutil.D3DFormatDXT4, "bc3_unorm", ddsutil.AlphaModePreMultiplied},
	} {
		dds, err := ddsutil.NewD3D(ddsutil.D3DParams{Width: 4, Height: 4, Format: tc.format})
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "in.dds")
		data, err := dds.Bytes()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := retag(path, path, tc.name, false); err != nil {
			t.Fatalf("%v: %v", tc.format, err)
		}
		after := readDds(t, path)
		if after.Header10 == nil || after.Header10.AlphaMode != tc.want {
			t.Errorf("%v: Header10 = %+v, want alpha mode %v", tc.format, after.Header10, tc.want)
		}
	}
}
