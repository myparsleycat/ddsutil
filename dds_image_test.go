package ddsutil

import (
	"errors"
	"image"
	"testing"
)

func TestSurfaceRgba8FromSubImage(t *testing.T) {
	parent := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for i := range parent.Pix {
		parent.Pix[i] = uint8(i)
	}
	sub := parent.SubImage(image.Rect(2, 4, 6, 8)).(*image.RGBA)
	surface := SurfaceRgba8FromImage(sub)
	if surface.Width != 4 || surface.Height != 4 || len(surface.Data) != 4*4*4 {
		t.Fatalf("surface %dx%d with %d bytes", surface.Width, surface.Height, len(surface.Data))
	}
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			want := sub.RGBAAt(2+x, 4+y)
			got := surface.Data[(y*4+x)*4:]
			if got[0] != want.R || got[1] != want.G || got[2] != want.B || got[3] != want.A {
				t.Fatalf("pixel (%d, %d) = %v, want %v", x, y, got[:4], want)
			}
		}
	}

	// Contiguous images are used without copying.
	if whole := SurfaceRgba8FromImage(parent); &whole.Data[0] != &parent.Pix[0] {
		t.Error("SurfaceRgba8FromImage copied a contiguous image")
	}
}

func TestIntoImageSharesData(t *testing.T) {
	surface := &SurfaceRgba8{Width: 2, Height: 3, Depth: 1, Layers: 1, Mipmaps: 1, Data: make([]uint8, 2*3*4)}
	img, err := surface.IntoImage()
	if err != nil {
		t.Fatal(err)
	}
	if img.Rect != image.Rect(0, 0, 2, 3) || &img.Pix[0] != &surface.Data[0] {
		t.Fatalf("IntoImage = %v, want a view of the surface data", img.Rect)
	}
}

func TestToImage3dMipmap(t *testing.T) {
	surface := &SurfaceRgba8{Width: 4, Height: 4, Depth: 4, Layers: 1, Mipmaps: 1, Data: make([]uint8, 4*4*4*4)}
	encoded, err := surface.Encode(Rgba8Unorm, QualityFast, MipmapsGeneratedAutomatic)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := encoded.DecodeRgba8()
	if err != nil {
		t.Fatal(err)
	}
	img, err := decoded.ToNRGBA(1)
	if err != nil {
		t.Fatal(err)
	}
	// Mipmap 1 is 2x2 with 2 depth slices stacked vertically.
	if img.Rect != image.Rect(0, 0, 2, 4) {
		t.Fatalf("ToNRGBA(1) rect = %v", img.Rect)
	}
}

func TestNRGBAFromDdsKeepsStraightAlpha(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for i := 0; i < len(src.Pix); i += 4 {
		copy(src.Pix[i:], []uint8{200, 100, 50, 64})
	}
	dds, err := DdsFromImage(src, Rgba8Unorm, QualityFast, MipmapsDisabled)
	if err != nil {
		t.Fatal(err)
	}
	img, err := NRGBAFromDds(dds, 0)
	if err != nil {
		t.Fatal(err)
	}
	if c := img.NRGBAAt(1, 1); c.R != 200 || c.G != 100 || c.B != 50 || c.A != 64 {
		t.Fatalf("NRGBAAt = %v, want {200 100 50 64}", c)
	}
}

func TestSurfaceRgba8FromShortImage(t *testing.T) {
	for _, stride := range []int{16, 32} {
		img := &image.RGBA{Rect: image.Rect(0, 0, 4, 4), Stride: stride, Pix: make([]uint8, 32)}
		_, err := DdsFromImage(img, Rgba8Unorm, QualityFast, MipmapsDisabled)
		var surfaceErr *SurfaceError
		if !errors.As(err, &surfaceErr) || surfaceErr.Kind != SurfaceErrorNotEnoughData {
			t.Errorf("stride %d: err = %v, want NotEnoughData", stride, err)
		}
	}
}
