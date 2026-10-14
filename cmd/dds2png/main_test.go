package main

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestConvert(t *testing.T) {
	for _, tc := range []struct {
		name              string
		layer, mip, slice uint32
		width, height     int
	}{
		{"dxt5_mip.dds", 0, 1, 0, 4, 4},
		{"cube.dds", 5, 0, 0, 4, 4},
		{"volume.dds", 0, 0, 1, 4, 4},
	} {
		output := filepath.Join(t.TempDir(), "out.png")
		if err := convert(filepath.Join("..", "..", "testdata", tc.name), output, tc.layer, tc.mip, tc.slice); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		file, err := os.Open(output)
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(file)
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
		if b := img.Bounds(); b.Dx() != tc.width || b.Dy() != tc.height {
			t.Errorf("%s: %v, want %dx%d", tc.name, b, tc.width, tc.height)
		}
	}

	output := filepath.Join(t.TempDir(), "out.png")
	if err := convert(filepath.Join("..", "..", "testdata", "bc1.dds"), output, 0, 5, 0); err == nil {
		t.Error("out of range mip accepted")
	}
}
