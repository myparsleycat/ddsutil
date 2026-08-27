package ddsutil

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

func mustNewD3D(t *testing.T, params NewD3dParams) *Dds {
	t.Helper()
	dds, err := NewD3D(params)
	if err != nil {
		t.Fatalf("NewD3D failed: %v", err)
	}
	return dds
}

// TestNewD3DHeaderBytes checks the exact byte layout of a minimal A8R8G8B8
// file against the DDS specification.
func TestNewD3DHeaderBytes(t *testing.T) {
	dds := mustNewD3D(t, NewD3dParams{
		Height: 4,
		Width:  4,
		Format: D3DFormatA8R8G8B8,
	})

	var buf bytes.Buffer
	if err := dds.Write(&buf); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	out := buf.Bytes()

	if got, want := len(out), 4+124+16*4; got != want {
		t.Fatalf("file size = %d, want %d", got, want)
	}

	r := bytes.NewReader(out)
	checkU32 := func(want uint32, name string) {
		t.Helper()
		var got uint32
		if err := binary.Read(r, binary.LittleEndian, &got); err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		if got != want {
			t.Errorf("%s = 0x%x, want 0x%x", name, got, want)
		}
	}
	checkU32(Magic, "magic")
	checkU32(124, "header size")
	// CAPS | HEIGHT | WIDTH | PIXELFORMAT | PITCH
	checkU32(0x1|0x2|0x4|0x1000|0x8, "flags")
	checkU32(4, "height")
	checkU32(4, "width")
	checkU32(16, "pitch") // (4*32+7)/8
	checkU32(0, "depth")
	checkU32(0, "mipmaps")
	for i := 0; i < 11; i++ {
		checkU32(0, "reserved1")
	}
	checkU32(32, "spf size")
	checkU32(0x1|0x40, "spf flags (ALPHA_PIXELS|RGB)")
	checkU32(0, "spf fourcc")
	checkU32(32, "spf bpp")
	checkU32(0x00ff0000, "spf r mask")
	checkU32(0x0000ff00, "spf g mask")
	checkU32(0x000000ff, "spf b mask")
	checkU32(0xff000000, "spf a mask")
	checkU32(0x1000, "caps (TEXTURE)")
	checkU32(0, "caps2")
	checkU32(0, "caps3")
	checkU32(0, "caps4")
	checkU32(0, "reserved2")

	// Data should be zeroed and 64 bytes.
	dds2, err := Read(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if len(dds2.Data) != 64 {
		t.Errorf("data size = %d, want 64", len(dds2.Data))
	}
	for i, b := range dds2.Data {
		if b != 0 {
			t.Fatalf("data[%d] = %d, want 0", i, b)
		}
	}
}

func TestRoundTripD3D(t *testing.T) {
	dds := mustNewD3D(t, NewD3dParams{
		Height: 8,
		Width:  8,
		Format: D3DFormatA8R8G8B8,
	})
	copy(dds.Data, bytes.Repeat([]byte{0xAB}, len(dds.Data)))

	var buf bytes.Buffer
	if err := dds.Write(&buf); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	dds2, err := Read(&buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if dds2.GetWidth() != 8 || dds2.GetHeight() != 8 {
		t.Errorf("dims = %dx%d, want 8x8", dds2.GetWidth(), dds2.GetHeight())
	}
	if dds2.Header10 != nil {
		t.Error("Header10 should be nil for D3D format")
	}
	f, ok := dds2.GetD3DFormat()
	if !ok || f != D3DFormatA8R8G8B8 {
		t.Errorf("GetD3DFormat = %v, %v; want A8R8G8B8, true", f, ok)
	}
	bpp, ok := dds2.GetBitsPerPixel()
	if !ok || bpp != 32 {
		t.Errorf("bpp = %d, %v; want 32, true", bpp, ok)
	}
	pitch, ok := dds2.GetPitch()
	if !ok || pitch != 32 {
		t.Errorf("pitch = %d, %v; want 32, true", pitch, ok)
	}
	if !bytes.Equal(dds2.Data, dds.Data) {
		t.Error("data mismatch after round trip")
	}
}

func TestNewD3DCompressedLinearSize(t *testing.T) {
	mml := uint32(3)
	dds := mustNewD3D(t, NewD3dParams{
		Height:       8,
		Width:        8,
		Format:       D3DFormatDXT5,
		MipmapLevels: &mml,
	})
	if dds.Header.LinearSize == nil || *dds.Header.LinearSize != 64 {
		t.Errorf("linear_size = %v, want 64", dds.Header.LinearSize)
	}
	if dds.Header.Pitch != nil {
		t.Errorf("pitch = %v, want None", *dds.Header.Pitch)
	}
	// mip chain: 64 + 16 + 16 = 96
	if len(dds.Data) != 96 {
		t.Errorf("data size = %d, want 96", len(dds.Data))
	}
	if dds.GetNumMipmapLevels() != 3 {
		t.Errorf("mipmap levels = %d, want 3", dds.GetNumMipmapLevels())
	}
}

func TestNewDXGIAndHeader10(t *testing.T) {
	dds, err := NewDXGI(NewDxgiParams{
		Height:            8,
		Width:             8,
		Format:            DxgiFormatBC1_UNorm,
		ResourceDimension: D3D10ResourceDimensionTexture2D,
		AlphaMode:         AlphaModeUnknown,
	})
	if err != nil {
		t.Fatalf("NewDXGI failed: %v", err)
	}
	if dds.Header10 == nil {
		t.Fatal("Header10 is nil")
	}
	if dds.Header10.DxgiFormat != DxgiFormatBC1_UNorm {
		t.Errorf("dxgi format = %v", dds.Header10.DxgiFormat)
	}
	if dds.Header10.ResourceDimension != D3D10ResourceDimensionTexture2D {
		t.Errorf("resource dimension = %v", dds.Header10.ResourceDimension)
	}
	if *dds.Header.SPF.FourCC != FourCCDX10 {
		t.Errorf("fourcc = %v, want DX10", dds.Header.SPF.FourCC)
	}
	// BC1 8x8: pitch = max(1,(8+3)/4)*8 = 16; rowheight 2 -> linear 32
	if dds.Header.LinearSize == nil || *dds.Header.LinearSize != 32 {
		t.Errorf("linear_size = %v, want 32", dds.Header.LinearSize)
	}
	if len(dds.Data) != 32 {
		t.Errorf("data size = %d, want 32", len(dds.Data))
	}

	var buf bytes.Buffer
	if err := dds.Write(&buf); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	dds2, err := Read(&buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if dds2.Header10 == nil || dds2.Header10.DxgiFormat != DxgiFormatBC1_UNorm {
		t.Error("Header10 not preserved through round trip")
	}
	if got := dds2.GetNumArrayLayers(); got != 1 {
		t.Errorf("array layers = %d, want 1", got)
	}
}

func TestCubemapArrayLayers(t *testing.T) {
	c2 := Caps2CUBEMAP | Caps2CUBEMAP_ALLFACES
	layers := uint32(6)
	dds, err := NewDXGI(NewDxgiParams{
		Height:            4,
		Width:             4,
		Format:            DxgiFormatR8G8B8A8_UNorm,
		ArrayLayers:       &layers,
		IsCubemap:         true,
		Caps2:             &c2,
		ResourceDimension: D3D10ResourceDimensionTexture2D,
		AlphaMode:         AlphaModeUnknown,
	})
	if err != nil {
		t.Fatalf("NewDXGI failed: %v", err)
	}
	// Note: NewDXGI stores the *number of cubemaps* (array_layers / 6) in
	// header10.array_size, and GetNumArrayLayers reads header10.array_size
	// when a DX10 header is present. So a single cubemap reports 1 here,
	// unlike the caps2-based path below.
	if got := dds.GetNumArrayLayers(); got != 1 {
		t.Errorf("array layers = %d, want 1", got)
	}
	if dds.Header10 == nil || dds.Header10.MiscFlag != MiscFlagTEXTURECUBE {
		t.Errorf("misc flag = %v, want TEXTURECUBE", dds.Header10)
	}
	// 6 layers * 64 bytes (4x4 RGBA8)
	if len(dds.Data) != 6*64 {
		t.Errorf("data size = %d, want %d", len(dds.Data), 6*64)
	}
	// Layer access
	data, err := dds.GetData(0)
	if err != nil {
		t.Fatalf("GetData(0) failed: %v", err)
	}
	if len(data) != 64 {
		t.Errorf("layer size = %d, want 64", len(data))
	}
	if _, err := dds.GetData(1); !errors.Is(err, ErrOutOfBounds) {
		t.Errorf("GetData(1) err = %v, want ErrOutOfBounds", err)
	}
}

// TestCubemapArrayLayersCaps2 checks the non-DX10 cubemap path, where the
// layer count is derived from the CUBEMAP caps2 flag.
func TestCubemapArrayLayersCaps2(t *testing.T) {
	c2 := Caps2CUBEMAP | Caps2CUBEMAP_ALLFACES
	dds := mustNewD3D(t, NewD3dParams{
		Height: 4,
		Width:  4,
		Format: D3DFormatA8R8G8B8,
		Caps2:  &c2,
	})
	if got := dds.GetNumArrayLayers(); got != 6 {
		t.Errorf("array layers = %d, want 6", got)
	}
	if len(dds.Data) != 64 {
		t.Errorf("data size = %d, want 64", len(dds.Data))
	}
	if _, err := dds.GetData(5); !errors.Is(err, ErrOutOfBounds) {
		t.Errorf("GetData(5) err = %v, want ErrOutOfBounds", err)
	}
}

func TestReadErrors(t *testing.T) {
	if _, err := Read(bytes.NewReader([]byte{0x01, 0x02, 0x03, 0x04})); !errors.Is(err, ErrBadMagicNumber) {
		t.Errorf("err = %v, want ErrBadMagicNumber", err)
	}
	if _, err := Read(bytes.NewReader([]byte{0x44, 0x53, 0x20})); !errors.Is(err, ErrShortFile) {
		t.Errorf("err = %v, want ErrShortFile", err)
	}
	// Truncated in the middle of the header (after the size field validates):
	// must surface as ErrShortFile.
	full := mustNewD3D(t, NewD3dParams{Height: 4, Width: 4, Format: D3DFormatA8R8G8B8})
	var fullBuf bytes.Buffer
	if err := full.Write(&fullBuf); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	truncated := fullBuf.Bytes()[:4+20]
	if _, err := Read(bytes.NewReader(truncated)); !errors.Is(err, ErrShortFile) {
		t.Errorf("err = %v, want ErrShortFile", err)
	}
	// Bad header struct size
	data := make([]byte, 4+128)
	binary.LittleEndian.PutUint32(data[0:4], Magic)
	binary.LittleEndian.PutUint32(data[4:8], 123)
	if _, err := Read(bytes.NewReader(data)); err == nil || !bytes.Contains([]byte(err.Error()), []byte("invalid field")) {
		t.Errorf("err = %v, want InvalidField error", err)
	}
}

func TestD3DFormatTryFromPixelFormat(t *testing.T) {
	pf := PixelFormatFromD3D(D3DFormatA8R8G8B8)
	got, ok := D3DFormatTryFromPixelFormat(&pf)
	if !ok || got != D3DFormatA8R8G8B8 {
		t.Errorf("got %v %v, want A8R8G8B8", got, ok)
	}

	pf = PixelFormatFromD3D(D3DFormatDXT5)
	got, ok = D3DFormatTryFromPixelFormat(&pf)
	if !ok || got != D3DFormatDXT5 {
		t.Errorf("got %v %v, want DXT5", got, ok)
	}

	// L8 does not round-trip through the pixel format table: the built
	// PixelFormat sets the RGB flag (not LUMINANCE), so the lookup finds
	// no match. Verify that behavior.
	pf = PixelFormatFromD3D(D3DFormatL8)
	if got, ok := D3DFormatTryFromPixelFormat(&pf); ok {
		t.Errorf("L8 pixel format unexpectedly matched %v", got)
	}

	// DX10 fourcc should not map to a D3D format.
	pf = PixelFormatFromDXGI(DxgiFormatR8G8B8A8_UNorm)
	if _, ok := D3DFormatTryFromPixelFormat(&pf); ok {
		t.Error("DX10 fourcc should not map to a D3D format")
	}
}

func TestDxgiFormatTryFromPixelFormat(t *testing.T) {
	pf := PixelFormat{Size: 32}
	fc := FourCC(FourCCDXT1)
	pf.FourCC = &fc
	pf.Flags = PixelFormatFlagsFOURCC
	got, ok := DxgiFormatTryFromPixelFormat(&pf)
	if !ok || got != DxgiFormatBC1_UNorm_sRGB {
		t.Errorf("got %v %v, want BC1_UNorm_sRGB", got, ok)
	}
}

func TestGetFormatPrefersDXGI(t *testing.T) {
	dds, err := NewDXGI(NewDxgiParams{
		Height:            4,
		Width:             4,
		Format:            DxgiFormatR8G8B8A8_UNorm,
		ResourceDimension: D3D10ResourceDimensionTexture2D,
		AlphaMode:         AlphaModeUnknown,
	})
	if err != nil {
		t.Fatalf("NewDXGI failed: %v", err)
	}
	f := dds.GetFormat()
	if f == nil {
		t.Fatal("GetFormat returned nil")
	}
	if _, ok := f.(DxgiFormat); !ok {
		t.Errorf("GetFormat returned %T, want DxgiFormat", f)
	}
	if f.RequiresExtension() {
		t.Error("R8G8B8A8_UNorm should not require DX10 extension")
	}
}

func TestGetPitchSpecialCases(t *testing.T) {
	p, ok := (D3DFormatR8G8_B8G8).GetPitch(5)
	if !ok || p != 12 { // ((5+1)>>1)*4
		t.Errorf("R8G8_B8G8 pitch(5) = %d %v, want 12", p, ok)
	}
	p, ok = (DxgiFormatBC7_UNorm).GetPitch(1)
	if !ok || p != 16 { // max(1, (1+3)/4) * 16
		t.Errorf("BC7 pitch(1) = %d %v, want 16", p, ok)
	}
}

func TestVolumeTextureDepth(t *testing.T) {
	depth := uint32(2)
	dds := mustNewD3D(t, NewD3dParams{
		Height: 4,
		Width:  4,
		Depth:  &depth,
		Format: D3DFormatA8R8G8B8,
	})
	if dds.GetDepth() != 2 {
		t.Errorf("depth = %d, want 2", dds.GetDepth())
	}
	if dds.Header.Depth == nil || *dds.Header.Depth != 2 {
		t.Error("header depth not set")
	}
	if !dds.Header.Flags.Contains(HeaderFlagsDEPTH) {
		t.Error("DEPTH flag not set")
	}
	if dds.Header.Caps&CapsCOMPLEX == 0 {
		t.Error("COMPLEX cap not set")
	}
	// pitch(16) * rowHeight(4) * depth(2) = 128
	if len(dds.Data) != 128 {
		t.Errorf("data size = %d, want 128", len(dds.Data))
	}
}

func TestDxgiFormatFromU32(t *testing.T) {
	if f, ok := DxgiFormatFromU32(28); !ok || f != DxgiFormatR8G8B8A8_UNorm {
		t.Errorf("got %v %v", f, ok)
	}
	if _, ok := DxgiFormatFromU32(116); ok {
		// 116 is not a defined DXGI value in this enum (gap between 115 and 130)
		t.Error("116 should not be a known DXGI format")
	}
}

func TestStringDebugOutput(t *testing.T) {
	dds := mustNewD3D(t, NewD3dParams{Height: 2, Width: 2, Format: D3DFormatA8R8G8B8})
	s := dds.String()
	if !bytes.Contains([]byte(s), []byte("A8R8G8B8")) {
		t.Errorf("debug output missing format name: %q", s)
	}
}
