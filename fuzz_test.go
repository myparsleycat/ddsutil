package ddsutil

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// maxFuzzPixels bounds the decoded size so fuzz inputs that claim huge
// dimensions are parsed but not decoded.
const maxFuzzPixels = 1 << 18

func addTestdataSeeds(f *testing.F) {
	paths, err := filepath.Glob(filepath.Join("testdata", "*.dds"))
	if err != nil {
		f.Fatal(err)
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(data)
	}
}

// FuzzParse checks that parsing, inspecting and decoding arbitrary input
// returns errors instead of panicking or hanging.
func FuzzParse(f *testing.F) {
	addTestdataSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		dds, err := Parse(data)
		if err != nil {
			return
		}
		_ = dds.String()
		_, _ = dds.ArrayStride()
		for layer := uint32(0); layer < 2; layer++ {
			_, _ = dds.LayerData(layer)
		}
		surface, err := SurfaceFromDds(dds)
		if err != nil {
			return
		}
		pixels := uint64(surface.Width) * uint64(surface.Height) * uint64(surface.Depth) * uint64(surface.Layers)
		if pixels > maxFuzzPixels {
			return
		}
		if decoded, err := surface.DecodeRgba8(); err == nil {
			_, _ = decoded.ToImage(0)
		}
		_, _ = surface.DecodeRgbaf32()
		if serialized, err := dds.Bytes(); err == nil {
			if _, err := Parse(serialized); err != nil {
				t.Fatalf("re-parsing serialized DDS: %v", err)
			}
		}
	})
}

// FuzzDdsReader checks the streaming reader on arbitrary input.
func FuzzDdsReader(f *testing.F) {
	addTestdataSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		reader, err := NewDdsReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return
		}
		m := reader.Metadata()
		if uint64(m.Width)*uint64(m.Height)*uint64(m.Depth) > maxFuzzPixels {
			return
		}
		last := m.Layers - 1
		lastMip := m.Mipmaps - 1
		_, _ = reader.ReadMip(last, lastMip)
		_, _ = reader.DecodeMipRgba8(0, 0)
		_, _ = reader.DecodeMipRgba8(last, lastMip)
		_, _ = reader.DecodeMipRgba8To(0, 0, (m.Width+1)/2, (m.Height+1)/2)
	})
}

// FuzzEncodeDecode checks that encoding arbitrary pixels to every format and
// decoding the result succeeds.
func FuzzEncodeDecode(f *testing.F) {
	f.Add([]byte{0, 255, 128, 64}, uint8(0), uint8(1))
	f.Fuzz(func(t *testing.T, pixels []byte, formatIndex, quality uint8) {
		formats := AllImageFormats()
		format := formats[int(formatIndex)%len(formats)]
		data := make([]byte, 8*8*4)
		for i := range data {
			if len(pixels) > 0 {
				data[i] = pixels[i%len(pixels)]
			}
		}
		surface := &SurfaceRgba8{Width: 8, Height: 8, Depth: 1, Layers: 1, Mipmaps: 1, Data: data}
		encoded, err := surface.Encode(format, Quality(quality%3), MipmapsGeneratedAutomatic)
		if err != nil {
			t.Fatalf("encode %v: %v", format, err)
		}
		if _, err := encoded.DecodeRgba8(); err != nil {
			t.Fatalf("decode %v: %v", format, err)
		}
	})
}
