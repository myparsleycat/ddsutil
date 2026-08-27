package ddsutil

import (
	"errors"
	"math"
	"testing"
)

func surfaceErr(err error) *SurfaceError {
	var se *SurfaceError
	if errors.As(err, &se) {
		return se
	}
	return nil
}

func TestDecodeSurfaceZeroSize(t *testing.T) {
	surface := &Surface{
		Width: 0, Height: 0, Depth: 0, Layers: 1, Mipmaps: 1,
		ImageFormat: Rgba8UnormSrgb, Data: []uint8{},
	}
	_, err := surface.DecodeRgba8()
	if err == nil || surfaceErr(err).Kind != SurfaceErrorZeroSizedSurface {
		t.Fatalf("expected ZeroSizedSurface, got %v", err)
	}
}

func TestDecodeSurfaceDimensionsOverflow(t *testing.T) {
	surface := &Surface{
		Width: ^uint32(0), Height: ^uint32(0), Depth: ^uint32(0), Layers: 1, Mipmaps: 1,
		ImageFormat: Rgba8UnormSrgb, Data: []uint8{},
	}
	_, err := surface.DecodeRgba8()
	if err == nil || surfaceErr(err).Kind != SurfaceErrorPixelCountWouldOverflow {
		t.Fatalf("expected PixelCountWouldOverflow, got %v", err)
	}
}

func TestDecodeSurfaceTooManyMipmaps(t *testing.T) {
	surface := &Surface{
		Width: 4, Height: 4, Depth: 1, Layers: 1, Mipmaps: 10,
		ImageFormat: Rgba8UnormSrgb, Data: make([]uint8, 4*4*4),
	}
	_, err := surface.DecodeRgba8()
	if err == nil || surfaceErr(err).Kind != SurfaceErrorUnexpectedMipmapCount || surfaceErr(err).MaxTotalMipmaps != 3 {
		t.Fatalf("expected UnexpectedMipmapCount max 3, got %v", err)
	}
}

func TestDecodeLayersMipmapsRgba8SingleMipmap(t *testing.T) {
	surface := &Surface{
		Width: 4, Height: 4, Depth: 1, Layers: 1, Mipmaps: 3,
		ImageFormat: Rgba8UnormSrgb, Data: make([]uint8, 512),
	}
	got, err := surface.DecodeLayersMipmapsRgba8(0, 1, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	want := &SurfaceRgba8{Width: 2, Height: 2, Depth: 1, Layers: 1, Mipmaps: 1, Data: make([]uint8, 2*2*4)}
	if got.Width != want.Width || got.Height != want.Height || got.Mipmaps != want.Mipmaps || len(got.Data) != len(want.Data) {
		t.Fatalf("mismatch: %+v", got)
	}
}

func TestDecodeLayersMipmapsRgba8NoMipmaps(t *testing.T) {
	surface := &Surface{
		Width: 4, Height: 4, Depth: 1, Layers: 1, Mipmaps: 1,
		ImageFormat: Rgba8UnormSrgb, Data: make([]uint8, 4*4*4),
	}
	got, err := surface.DecodeLayersMipmapsRgba8(0, 1, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Data) != 0 || got.Mipmaps != 1 || got.Width != 4 {
		t.Fatalf("mismatch: %+v", got)
	}
}

func TestEncodeSurfaceIntegralDimensions(t *testing.T) {
	// It's ok for mipmaps to not be divisible by the block width.
	surface := &SurfaceRgba8{
		Width: 12, Height: 12, Depth: 1, Layers: 1, Mipmaps: 1,
		Data: make([]uint8, 12*12*4),
	}
	got, err := surface.Encode(BC7RgbaUnormSrgb, QualityFast, MipmapsGeneratedAutomatic)
	if err != nil {
		t.Fatal(err)
	}
	if got.Width != 12 || got.Height != 12 || got.Depth != 1 || got.Layers != 1 || got.Mipmaps != 4 {
		t.Fatalf("mismatch: %+v", got)
	}
	if got.ImageFormat != BC7RgbaUnormSrgb {
		t.Fatalf("format %v", got.ImageFormat)
	}
	// Each mipmap must be at least 1 block in size.
	if len(got.Data) != (9+4+1+1)*16 {
		t.Fatalf("data len %d", len(got.Data))
	}
}

func TestEncodeSurfaceCubeMipmaps(t *testing.T) {
	surface := &SurfaceRgba8{
		Width: 4, Height: 4, Depth: 1, Layers: 6, Mipmaps: 3,
		Data: make([]uint8, (4*4+2*2+1*1)*6*4),
	}
	got, err := surface.Encode(BC7RgbaUnormSrgb, QualityFast, MipmapsGeneratedAutomatic)
	if err != nil {
		t.Fatal(err)
	}
	if got.Mipmaps != 3 || len(got.Data) != 3*16*6 {
		t.Fatalf("mismatch: %+v len %d", got, len(got.Data))
	}
}

func TestEncodeSurfaceDisabledMipmaps(t *testing.T) {
	surface := &SurfaceRgba8{
		Width: 4, Height: 4, Depth: 1, Layers: 1, Mipmaps: 3,
		Data: make([]uint8, 64+16+4),
	}
	got, err := surface.Encode(BC7RgbaUnormSrgb, QualityFast, MipmapsDisabled)
	if err != nil {
		t.Fatal(err)
	}
	if got.Mipmaps != 1 || len(got.Data) != 16 {
		t.Fatalf("mismatch: %+v len %d", got, len(got.Data))
	}
}

func TestEncodeSurfaceMipmapsFromSurface(t *testing.T) {
	surface := &SurfaceRgba8{
		Width: 4, Height: 4, Depth: 1, Layers: 1, Mipmaps: 2,
		Data: make([]uint8, 64+16),
	}
	got, err := surface.Encode(BC7RgbaUnormSrgb, QualityFast, MipmapsFromSurface)
	if err != nil {
		t.Fatal(err)
	}
	if got.Mipmaps != 2 || len(got.Data) != 16*2 {
		t.Fatalf("mismatch: %+v len %d", got, len(got.Data))
	}
}

func TestEncodeSurfaceNonIntegralDimensions(t *testing.T) {
	// This should succeed with appropriate padding.
	surface := &SurfaceRgba8{
		Width: 3, Height: 5, Depth: 1, Layers: 1, Mipmaps: 1,
		Data: make([]uint8, 256),
	}
	got, err := surface.Encode(BC7RgbaUnormSrgb, QualityFast, MipmapsGeneratedAutomatic)
	if err != nil {
		t.Fatal(err)
	}
	if got.Width != 3 || got.Height != 5 || got.Mipmaps != 3 {
		t.Fatalf("mismatch: %+v", got)
	}
	// Each mipmap must have an integral size in blocks.
	if len(got.Data) != (2+2)*16 {
		t.Fatalf("data len %d", len(got.Data))
	}
}

func TestEncodeSurfaceZeroSize(t *testing.T) {
	surface := &SurfaceRgba8{
		Width: 0, Height: 0, Depth: 0, Layers: 1, Mipmaps: 1, Data: []uint8{},
	}
	_, err := surface.Encode(BC7RgbaUnormSrgb, QualityFast, MipmapsGeneratedAutomatic)
	if err == nil || surfaceErr(err).Kind != SurfaceErrorZeroSizedSurface {
		t.Fatalf("expected ZeroSizedSurface, got %v", err)
	}
}

func TestEncodeSurfaceFloat322dMipmaps(t *testing.T) {
	data := make([]float32, 36)
	for i := range data {
		data[i] = float32(i)
	}
	surface := &SurfaceRgba32Float{
		Width: 3, Height: 3, Depth: 1, Layers: 1, Mipmaps: 1, Data: data,
	}
	got, err := surface.Encode(Rgba32Float, QualityFast, MipmapsGeneratedAutomatic)
	if err != nil {
		t.Fatal(err)
	}
	want := []float32{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17,
		18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35,
		8, 9, 10, 11,
	}
	if got.Mipmaps != 2 {
		t.Fatalf("mipmaps %d", got.Mipmaps)
	}
	if len(got.Data) != len(want)*4 {
		t.Fatalf("data len %d want %d", len(got.Data), len(want)*4)
	}
	for i, f := range want {
		gotF := math.Float32frombits(uint32(got.Data[i*4]) | uint32(got.Data[i*4+1])<<8 | uint32(got.Data[i*4+2])<<16 | uint32(got.Data[i*4+3])<<24)
		if gotF != f {
			t.Fatalf("element %d: got %v want %v", i, gotF, f)
		}
	}
}

func TestEncodeSurfaceFloat323dMipmaps(t *testing.T) {
	data := make([]float32, 108)
	for i := range data {
		data[i] = float32(i)
	}
	surface := &SurfaceRgba32Float{
		Width: 3, Height: 3, Depth: 3, Layers: 1, Mipmaps: 1, Data: data,
	}
	got, err := surface.Encode(Rgba32Float, QualityFast, MipmapsGeneratedAutomatic)
	if err != nil {
		t.Fatal(err)
	}
	if got.Mipmaps != 2 {
		t.Fatalf("mipmaps %d", got.Mipmaps)
	}
	// 27 base pixels + 1 downsample pixel.
	if len(got.Data) != 28*16 {
		t.Fatalf("data len %d", len(got.Data))
	}
	// The last pixel is the averaged 2x2x2 block of pixels 0,1,3,4,9,10,12,13
	// per depth level pair (z=0..1).
	last := pixelF32(got.Data, 27)
	want := [4]float32{26, 27, 28, 29}
	if last != want {
		t.Fatalf("last pixel %v want %v", last, want)
	}
}

func pixelF32(data []uint8, index int) [4]float32 {
	base := index * 16
	return [4]float32{
		math.Float32frombits(uint32(data[base]) | uint32(data[base+1])<<8 | uint32(data[base+2])<<16 | uint32(data[base+3])<<24),
		math.Float32frombits(uint32(data[base+4]) | uint32(data[base+5])<<8 | uint32(data[base+6])<<16 | uint32(data[base+7])<<24),
		math.Float32frombits(uint32(data[base+8]) | uint32(data[base+9])<<8 | uint32(data[base+10])<<16 | uint32(data[base+11])<<24),
		math.Float32frombits(uint32(data[base+12]) | uint32(data[base+13])<<8 | uint32(data[base+14])<<16 | uint32(data[base+15])<<24),
	}
}

func TestEncodeSurfaceFloat32CubeMipmaps(t *testing.T) {
	data := make([]float32, 216)
	for i := range data {
		data[i] = float32(i)
	}
	surface := &SurfaceRgba32Float{
		Width: 3, Height: 3, Depth: 1, Layers: 6, Mipmaps: 1, Data: data,
	}
	got, err := surface.Encode(Rgba32Float, QualityFast, MipmapsGeneratedAutomatic)
	if err != nil {
		t.Fatal(err)
	}
	if got.Mipmaps != 2 || got.Layers != 6 {
		t.Fatalf("mipmaps %d layers %d", got.Mipmaps, got.Layers)
	}
	// 10 pixels per layer (9 + 1 downsample).
	if len(got.Data) != 10*16*6 {
		t.Fatalf("data len %d", len(got.Data))
	}
}

func TestPadMipmapRgba(t *testing.T) {
	// 1x1 to 1x1: no padding.
	data := []uint8{1, 2, 3, 4}
	if got := padMipmapRgba(1, 1, 1, 1, 1, 1, data); len(got) != 4 || got[0] != 1 {
		t.Fatalf("pad 1x1: %v", got)
	}
	// 1x1 to 2x2: zero padding.
	got := padMipmapRgba(1, 1, 1, 2, 2, 1, data)
	want := []uint8{1, 2, 3, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	if len(got) != len(want) {
		t.Fatalf("pad 2x2 len %d", len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("pad 2x2 element %d: %d want %d", i, got[i], want[i])
		}
	}
}

func TestPhysicalDimensions(t *testing.T) {
	if w, h, d := physicalDimensions(2, 3, 1, [3]uint32{4, 5, 6}); w != 4 || h != 5 || d != 6 {
		t.Fatalf("physicalDimensions: %d %d %d", w, h, d)
	}
	if w, _, _ := physicalDimensions(8, 8, 1, [3]uint32{4, 4, 1}); w != 8 {
		t.Fatalf("8x8 -> %d", w)
	}
	if w, _, _ := physicalDimensions(2, 2, 1, [3]uint32{4, 4, 1}); w != 4 {
		t.Fatalf("2x2 -> %d", w)
	}
	if w, _, _ := physicalDimensions(1, 1, 1, [3]uint32{4, 4, 1}); w != 4 {
		t.Fatalf("1x1 -> %d", w)
	}
}

func TestDecodeEncodeAllFormats(t *testing.T) {
	for _, format := range AllImageFormats() {
		surface := &SurfaceRgba8{
			Width: 4, Height: 4, Depth: 1, Layers: 1, Mipmaps: 1,
			Data: make([]uint8, 4*4*4),
		}
		if _, err := surface.Encode(format, QualityNormal, MipmapsGeneratedAutomatic); err != nil {
			t.Errorf("encode %v: %v", format, err)
		}
	}
}

func TestDecodeEncodeAllFormatsF32(t *testing.T) {
	for _, format := range AllImageFormats() {
		surface := &SurfaceRgba32Float{
			Width: 4, Height: 4, Depth: 1, Layers: 1, Mipmaps: 1,
			Data: make([]float32, 4*4*4),
		}
		if _, err := surface.Encode(format, QualityNormal, MipmapsGeneratedAutomatic); err != nil {
			t.Errorf("encode f32 %v: %v", format, err)
		}
	}
}
