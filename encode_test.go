package ddsutil

import (
	"bytes"
	"errors"
	"testing"
)

func TestEncodeWithBC7Fast(t *testing.T) {
	source := &SurfaceRgba8{
		Width: 5, Height: 3, Depth: 1, Layers: 1, Mipmaps: 1,
		Data: bytes.Repeat([]byte{64, 128, 192, 255}, 5*3),
	}
	const format = BC7RgbaUnormSrgb
	mipmaps := MipmapsGeneratedExact(2)
	fallback, err := source.Encode(format, QualityFast, mipmaps)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("handles the first mip and declines the second", func(t *testing.T) {
		var calls int
		marker := bytes.Repeat([]byte{0xA5}, 32)
		got, err := source.EncodeWithBC7Fast(format, QualityFast, mipmaps,
			func(width, height uint32, rgba, dst []byte) (bool, error) {
				calls++
				switch calls {
				case 1:
					if width != 8 || height != 4 || len(rgba) != 128 || len(dst) != 32 {
						t.Fatalf("mip 0: got %dx%d with %d input and %d output bytes", width, height, len(rgba), len(dst))
					}
					copy(dst, marker)
					return true, nil
				case 2:
					if width != 4 || height != 4 || len(rgba) != 64 || len(dst) != 16 {
						t.Fatalf("mip 1: got %dx%d with %d input and %d output bytes", width, height, len(rgba), len(dst))
					}
					return false, nil
				default:
					t.Fatalf("callback called %d times", calls)
					return false, nil
				}
			})
		if err != nil {
			t.Fatal(err)
		}
		if calls != 2 || got.Mipmaps != 2 || len(got.Data) != len(fallback.Data) {
			t.Fatalf("got %d calls, %d mip levels and %d output bytes", calls, got.Mipmaps, len(got.Data))
		}
		if !bytes.Equal(got.Data[:32], marker) || !bytes.Equal(got.Data[32:], fallback.Data[32:]) {
			t.Fatal("encoded data does not contain the callback output followed by the Go-encoded mip")
		}
	})

	t.Run("propagates encoder errors", func(t *testing.T) {
		encodeErr := errors.New("accelerator failed")
		got, err := source.EncodeWithBC7Fast(format, QualityFast, mipmaps,
			func(_, _ uint32, _, _ []byte) (bool, error) {
				return false, encodeErr
			})
		if got != nil || !errors.Is(err, encodeErr) {
			t.Fatalf("got surface %v and error %v, want no surface and %v", got, err, encodeErr)
		}
	})
}
