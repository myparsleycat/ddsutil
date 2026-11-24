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

func TestEncodeWithBlockEncoder(t *testing.T) {
	source := &SurfaceRgba8{
		Width: 4, Height: 4, Depth: 1, Layers: 1, Mipmaps: 1,
		Data: bytes.Repeat([]byte{0, 127, 128, 255}, 16),
	}

	t.Run("offers signed formats as snorm8 bit patterns", func(t *testing.T) {
		want, err := source.Encode(BC5RgSnorm, QualitySlow, MipmapsDisabled)
		if err != nil {
			t.Fatal(err)
		}
		var calls int
		got, err := source.EncodeWithBlockEncoder(BC5RgSnorm, QualitySlow, MipmapsDisabled,
			func(format ImageFormat, quality Quality, width, height uint32, data, dst []byte) (bool, error) {
				calls++
				if format != BC5RgSnorm || quality != QualitySlow || width != 4 || height != 4 || len(dst) != 16 {
					t.Fatalf("got %v at quality %v, %dx%d, %d output bytes", format, quality, width, height, len(dst))
				}
				if !bytes.Equal(data, bytes.Repeat([]byte{129, 0, 0, 127}, 16)) {
					t.Fatalf("data = %v", data[:4])
				}
				return false, nil
			})
		if err != nil || calls != 1 || !bytes.Equal(got.Data, want.Data) {
			t.Fatalf("got %d calls and error %v, or output differs from the built-in encoder", calls, err)
		}
	})

	t.Run("does not offer BC6H", func(t *testing.T) {
		_, err := source.EncodeWithBlockEncoder(BC6hRgbUfloat, QualityFast, MipmapsDisabled,
			func(ImageFormat, Quality, uint32, uint32, []byte, []byte) (bool, error) {
				t.Fatal("encoder called for BC6H")
				return false, nil
			})
		if err != nil {
			t.Fatal(err)
		}
	})
}
