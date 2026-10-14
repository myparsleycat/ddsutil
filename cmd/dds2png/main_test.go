package main

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/myparsleycat/ddsutil"
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

func TestConvertAlphaMode(t *testing.T) {
	for _, tc := range []struct {
		mode  ddsutil.AlphaMode
		pixel [4]byte
		want  [4]byte
	}{
		{ddsutil.AlphaModeStraight, [4]byte{128, 0, 0, 128}, [4]byte{128, 0, 0, 128}},
		{ddsutil.AlphaModeUnknown, [4]byte{128, 0, 0, 128}, [4]byte{128, 0, 0, 128}},
		{ddsutil.AlphaModePreMultiplied, [4]byte{128, 0, 0, 128}, [4]byte{255, 0, 0, 128}},
		{ddsutil.AlphaModePreMultiplied, [4]byte{64, 32, 200, 128}, [4]byte{128, 64, 255, 128}},
		{ddsutil.AlphaModePreMultiplied, [4]byte{10, 20, 30, 0}, [4]byte{10, 20, 30, 0}},
	} {
		dds, err := ddsutil.NewDxgi(ddsutil.DxgiParams{Width: 1, Height: 1, Format: ddsutil.DxgiFormatR8G8B8A8_UNorm, AlphaMode: tc.mode})
		if err != nil {
			t.Fatal(err)
		}
		copy(dds.Data, tc.pixel[:])
		dir := t.TempDir()
		input := filepath.Join(dir, "in.dds")
		file, err := os.Create(input)
		if err != nil {
			t.Fatal(err)
		}
		if err := dds.Write(file); err != nil {
			t.Fatal(err)
		}
		file.Close()

		output := filepath.Join(dir, "out.png")
		if err := convert(input, output, 0, 0, 0); err != nil {
			t.Fatal(err)
		}
		file, err = os.Open(output)
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(file)
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
		nrgba, ok := img.(*image.NRGBA)
		if !ok {
			t.Fatalf("%v: decoded %T, want *image.NRGBA", tc.mode, img)
		}
		if got := [4]byte(nrgba.Pix[:4]); got != tc.want {
			t.Errorf("%v %v: pixel = %v, want %v", tc.mode, tc.pixel, got, tc.want)
		}
	}
}
