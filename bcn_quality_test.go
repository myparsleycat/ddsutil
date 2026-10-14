package ddsutil

import (
	"math"
	"math/rand"
	"testing"
)

// testPattern returns a deterministic RGBA8 image with smooth gradients and
// noise in every channel, which exercises all encoder search paths.
func testPattern(n uint32) *SurfaceRgba8 {
	r := rand.New(rand.NewSource(2))
	data := make([]byte, n*n*4)
	for y := uint32(0); y < n; y++ {
		for x := uint32(0); x < n; x++ {
			i := (y*n + x) * 4
			fx, fy := float64(x)/float64(n), float64(y)/float64(n)
			values := [4]float64{
				128 + 100*math.Sin(fx*20+fy*3),
				128 + 90*math.Cos(fy*17-fx*5),
				128 + 80*math.Sin((fx+fy)*11),
				200 + 55*math.Sin(fx*7),
			}
			for c, v := range values {
				v += r.NormFloat64() * 6
				data[i+uint32(c)] = byte(math.Max(0, math.Min(255, v)))
			}
		}
	}
	return &SurfaceRgba8{Width: n, Height: n, Depth: 1, Layers: 1, Mipmaps: 1, Data: data}
}

// encodedPsnr encodes surface and returns the PSNR of the first channels
// channels of the decoded result.
func encodedPsnr(t testing.TB, surface *SurfaceRgba8, format ImageFormat, quality Quality, channels int) float64 {
	t.Helper()
	encoded, err := surface.Encode(format, quality, MipmapsDisabled)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := encoded.DecodeRgba8()
	if err != nil {
		t.Fatal(err)
	}
	var squaredError float64
	for i := 0; i < len(surface.Data); i += 4 {
		for c := 0; c < channels; c++ {
			d := float64(decoded.Data[i+c]) - float64(surface.Data[i+c])
			squaredError += d * d
		}
	}
	mse := squaredError / float64(len(surface.Data)/4*channels)
	return 10 * math.Log10(255*255/mse)
}

// TestEncodeQualityLevels checks that higher quality levels do not produce
// worse results than lower ones.
func TestEncodeQualityLevels(t *testing.T) {
	surface := testPattern(64)
	formats := []struct {
		format   ImageFormat
		channels int
	}{
		{BC1RgbaUnorm, 3},
		{BC2RgbaUnorm, 4},
		{BC3RgbaUnorm, 4},
		{BC4RUnorm, 1},
		{BC4RSnorm, 1},
		{BC5RgUnorm, 2},
		{BC5RgSnorm, 2},
		{BC6hRgbUfloat, 3},
		{BC7RgbaUnorm, 4},
	}
	for _, tc := range formats {
		tc := tc
		t.Run(tc.format.String(), func(t *testing.T) {
			fast := encodedPsnr(t, surface, tc.format, QualityFast, tc.channels)
			normal := encodedPsnr(t, surface, tc.format, QualityNormal, tc.channels)
			slow := encodedPsnr(t, surface, tc.format, QualitySlow, tc.channels)
			t.Logf("PSNR fast %.2f dB, normal %.2f dB, slow %.2f dB", fast, normal, slow)
			const tolerance = 0.05
			if normal < fast-tolerance || slow < normal-tolerance {
				t.Errorf("quality decreases: fast %.2f dB, normal %.2f dB, slow %.2f dB", fast, normal, slow)
			}
			if fast < 22 {
				t.Errorf("fast PSNR %.2f dB is unexpectedly low", fast)
			}
		})
	}
}

// TestEncodeParallelDeterministic checks that parallel encoding produces the
// same output as encoding each block row on its own.
func TestEncodeParallelDeterministic(t *testing.T) {
	surface := testPattern(128)
	for _, format := range []ImageFormat{BC1RgbaUnorm, BC7RgbaUnorm} {
		whole, err := surface.Encode(format, QualityNormal, MipmapsDisabled)
		if err != nil {
			t.Fatal(err)
		}
		rowBytes := 128 / 4 * format.BlockSizeInBytes()
		for by := 0; by < 128/4; by++ {
			row := &SurfaceRgba8{Width: 128, Height: 4, Depth: 1, Layers: 1, Mipmaps: 1, Data: surface.Data[by*4*128*4 : (by+1)*4*128*4]}
			encoded, err := row.Encode(format, QualityNormal, MipmapsDisabled)
			if err != nil {
				t.Fatal(err)
			}
			if string(encoded.Data) != string(whole.Data[by*rowBytes:(by+1)*rowBytes]) {
				t.Fatalf("%v: block row %d differs", format, by)
			}
		}
	}
}

func BenchmarkEncode(b *testing.B) {
	surface := testPattern(512)
	for _, format := range []ImageFormat{BC1RgbaUnorm, BC3RgbaUnorm, BC4RUnorm, BC5RgUnorm, BC6hRgbUfloat, BC7RgbaUnorm} {
		for _, quality := range []Quality{QualityFast, QualityNormal, QualitySlow} {
			b.Run(format.String()+"/"+[]string{"fast", "normal", "slow"}[quality], func(b *testing.B) {
				b.SetBytes(int64(len(surface.Data)))
				for i := 0; i < b.N; i++ {
					if _, err := surface.Encode(format, quality, MipmapsDisabled); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

func BenchmarkDecode(b *testing.B) {
	surface := testPattern(2048)
	for _, format := range []ImageFormat{BC1RgbaUnorm, BC7RgbaUnorm} {
		encoded, err := surface.Encode(format, QualityFast, MipmapsGeneratedAutomatic)
		if err != nil {
			b.Fatal(err)
		}
		b.Run(format.String(), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := encoded.DecodeRgba8(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkGenerateMipmaps(b *testing.B) {
	surface := testPattern(2048)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := surface.Encode(Rgba8Unorm, QualityFast, MipmapsGeneratedAutomatic); err != nil {
			b.Fatal(err)
		}
	}
}
