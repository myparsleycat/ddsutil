package ddsutil

import (
	"encoding/binary"
	"testing"
)

// rawLegacy builds a minimal legacy DDS file (no DX10 header) with the given
// pixel format fields, for format detection tests.
func rawLegacy(spfFlags, fourcc, bitcount, r, g, b, a uint32, data ...byte) []byte {
	out := make([]byte, 128)
	copy(out, []byte("DDS "))
	putLE(out, 4, 124)                // header size
	putLE(out, 8, 0x1|0x2|0x4|0x1000) // CAPS|HEIGHT|WIDTH|PIXELFORMAT
	putLE(out, 12, 4)                 // height
	putLE(out, 16, 4)                 // width
	putLE(out, 76, 32)                // spf size
	putLE(out, 80, spfFlags)
	putLE(out, 84, fourcc)
	putLE(out, 88, bitcount)
	putLE(out, 92, r)
	putLE(out, 96, g)
	putLE(out, 100, b)
	putLE(out, 104, a)
	putLE(out, 108, 0x1000) // caps TEXTURE
	return append(out, data...)
}

// rawDX10 builds a minimal DX10 DDS file with the given extension fields.
func rawDX10(format uint32, dim uint32, misc uint32, array uint32, miscFlags2 uint32, data ...byte) []byte {
	out := rawLegacy(uint32(PixelFormatFlagsFOURCC), uint32(FourCCDX10), 0, 0, 0, 0, 0)
	dx10 := make([]byte, 20)
	putLE(dx10, 0, format)
	putLE(dx10, 4, dim)
	putLE(dx10, 8, misc)
	putLE(dx10, 12, array)
	putLE(dx10, 16, miscFlags2)
	out = append(out, dx10...)
	return append(out, data...)
}

func putLE(b []byte, off int, v uint32) { binary.LittleEndian.PutUint32(b[off:], v) }

// decodeBlockDds builds a 4x4 DDS file in memory containing the given
// compressed block data and decodes mip level 0 to RGBA8.
func decodeBlockDds(t *testing.T, format DxgiFormat, block []byte) []uint8 {
	t.Helper()
	d, err := NewDXGI(NewDxgiParams{
		Height:            4,
		Width:             4,
		Format:            format,
		ResourceDimension: D3D10ResourceDimensionTexture2D,
	})
	if err != nil {
		t.Fatalf("NewDXGI: %v", err)
	}
	d.Data = block
	img, err := ImageFromDds(d, 0)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if img.Rect.Dx() != 4 || img.Rect.Dy() != 4 {
		t.Fatalf("decoded dimensions %v", img.Rect)
	}
	return img.Pix
}

// pixelAt returns the RGBA8 values of the pixel at (x, y) in a 4x4 surface.
func pixelAt(pix []uint8, x, y int) [4]uint8 {
	i := (y*4 + x) * 4
	return [4]uint8{pix[i], pix[i+1], pix[i+2], pix[i+3]}
}

// solid4x4 returns a 4x4 pixel grid filled with v.
func solid4x4(v [4]uint8) [4][4][4]uint8 {
	var out [4][4][4]uint8
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			out[y][x] = v
		}
	}
	return out
}

// TestDecodeKnownAnswerBlocks verifies block decoding against expected values
// derived from the BCn specifications, using hand-crafted compressed blocks.
// The suite is fully self-contained: no external reference data is required.
func TestDecodeKnownAnswerBlocks(t *testing.T) {
	check := func(t *testing.T, name string, format DxgiFormat, block []byte, want [4][4][4]uint8) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			pix := decodeBlockDds(t, format, block)
			for y := 0; y < 4; y++ {
				for x := 0; x < 4; x++ {
					if got := pixelAt(pix, x, y); got != want[y][x] {
						t.Fatalf("pixel (%d,%d): got %v want %v", x, y, got, want[y][x])
					}
				}
			}
		})
	}

	// BC1: c0 = white (0xFFFF), c1 = black (0x0000) selects the 4-color
	// mode with palette white, black, (170,170,170), (85,85,85). Row 0 uses
	// indices 0,1,2,3; all other pixels use index 0.
	{
		block := []byte{
			0xFF, 0xFF, 0x00, 0x00, // c0, c1
			0xE4, 0x00, 0x00, 0x00, // indices: 0,1,2,3, then 0
		}
		want := solid4x4([4]uint8{255, 255, 255, 255})
		want[0][1] = [4]uint8{0, 0, 0, 255}
		want[0][2] = [4]uint8{170, 170, 170, 255}
		want[0][3] = [4]uint8{85, 85, 85, 255}
		check(t, "bc1", DxgiFormatBC1_UNorm, block, want)
	}

	// BC2: 4-bit alpha (0xF=255, 0x0=0, 0x8=136, 0x1=17 on row 0) plus a
	// black BC1 color block with all indices 0.
	{
		block := []byte{
			0x0F, 0x18, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, // alpha
			0x00, 0x00, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, // color
		}
		want := solid4x4([4]uint8{0, 0, 0, 255})
		want[0][1] = [4]uint8{0, 0, 0, 0}
		want[0][2] = [4]uint8{0, 0, 0, 136}
		want[0][3] = [4]uint8{0, 0, 0, 17}
		check(t, "bc2", DxgiFormatBC2_UNorm, block, want)
	}

	// BC3: 8-bit alpha endpoints 0xFF/0x00 give the palette
	// [255, 0, 218, 182, 145, 109, 73, 36]; row 0 uses indices 0,1,2,3.
	// The color block is black with all indices 0.
	{
		block := []byte{
			0xFF, 0x00, 0x88, 0x06, 0x00, 0x00, 0x00, 0x00, // alpha
			0x00, 0x00, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, // color
		}
		want := solid4x4([4]uint8{0, 0, 0, 255})
		want[0][1] = [4]uint8{0, 0, 0, 0}
		want[0][2] = [4]uint8{0, 0, 0, 218}
		want[0][3] = [4]uint8{0, 0, 0, 182}
		check(t, "bc3", DxgiFormatBC3_UNorm, block, want)
	}

	// BC4: same endpoint/index layout as the BC3 alpha block, decoded as
	// grayscale. The palette is [255, 0, 219, 182, 146, 109, 73, 36].
	{
		block := []byte{
			0xFF, 0x00, 0x88, 0x06, 0x00, 0x00, 0x00, 0x00,
		}
		want := solid4x4([4]uint8{255, 255, 255, 255})
		want[0][1] = [4]uint8{0, 0, 0, 255}
		want[0][2] = [4]uint8{219, 219, 219, 255}
		want[0][3] = [4]uint8{182, 182, 182, 255}
		check(t, "bc4", DxgiFormatBC4_UNorm, block, want)
	}

	// BC5: R channel like BC4; G channel endpoints 0x00/0xFF give the
	// palette [0, 255, 51, 102, 153, 204, 0, 255] with indices 1,2,3,4 on
	// row 0.
	{
		block := []byte{
			0xFF, 0x00, 0x88, 0x06, 0x00, 0x00, 0x00, 0x00, // R
			0x00, 0xFF, 0xD1, 0x08, 0x00, 0x00, 0x00, 0x00, // G
		}
		want := solid4x4([4]uint8{255, 0, 0, 255})
		want[0][0] = [4]uint8{255, 255, 0, 255}
		want[0][1] = [4]uint8{0, 51, 0, 255}
		want[0][2] = [4]uint8{219, 102, 0, 255}
		want[0][3] = [4]uint8{182, 153, 0, 255}
		check(t, "bc5", DxgiFormatBC5_UNorm, block, want)
	}

	// BC6H: mode 11 (explicit 10-bit endpoints). Endpoints 0 and 1023
	// unquantize to 0 and 0xFFFF, which decode to black and white. Row 0
	// uses indices 0,15,15,15.
	{
		block := []byte{
			0x03, 0x00, 0x00, 0x00, 0xF8, 0xFF, 0xFF, 0xFF, // mode + endpoints
			0xF1, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // indices
		}
		want := solid4x4([4]uint8{0, 0, 0, 255})
		want[0][1] = [4]uint8{255, 255, 255, 255}
		want[0][2] = [4]uint8{255, 255, 255, 255}
		want[0][3] = [4]uint8{255, 255, 255, 255}
		check(t, "bc6h", DxgiFormatBC6H_UF16, block, want)
	}

	// BC7: mode 6 (7-bit RGBA endpoints with p-bits). Endpoints are stored
	// channel-major (R0,R1,G0,G1,B0,B1,A0,A1); only B1 and A1 are 127, so
	// endpoint 1 decodes to (0,0,254,254) and endpoint 0 to (0,0,0,0).
	// Row 0 uses indices 0,15,15,15.
	{
		block := []byte{
			0x40, 0x00, 0x00, 0x00, 0x00, 0xFC, 0x01, 0x7F, // mode + endpoints
			0xF0, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // indices
		}
		want := solid4x4([4]uint8{0, 0, 0, 0})
		want[0][1] = [4]uint8{0, 0, 254, 254}
		want[0][2] = [4]uint8{0, 0, 254, 254}
		want[0][3] = [4]uint8{0, 0, 254, 254}
		check(t, "bc7", DxgiFormatBC7_UNorm, block, want)
	}
}

func TestDdsFormatDetection(t *testing.T) {
	// Build DDS files in memory (no external data) and check that the
	// format is detected correctly for DX10 and legacy FourCC headers.
	cases := []struct {
		name   string
		raw    []byte
		format ImageFormat
	}{
		{"bc1_dx10", rawDX10(uint32(DxgiFormatBC1_UNorm), 3, 0, 1, 0), BC1RgbaUnorm},
		{"bc2_dx10", rawDX10(uint32(DxgiFormatBC2_UNorm), 3, 0, 1, 0), BC2RgbaUnorm},
		{"bc3_dx10", rawDX10(uint32(DxgiFormatBC3_UNorm), 3, 0, 1, 0), BC3RgbaUnorm},
		{"bc4_dx10", rawDX10(uint32(DxgiFormatBC4_UNorm), 3, 0, 1, 0), BC4RUnorm},
		{"bc4_signed_dx10", rawDX10(uint32(DxgiFormatBC4_SNorm), 3, 0, 1, 0), BC4RSnorm},
		{"bc5_dx10", rawDX10(uint32(DxgiFormatBC5_UNorm), 3, 0, 1, 0), BC5RgUnorm},
		{"bc5_signed_dx10", rawDX10(uint32(DxgiFormatBC5_SNorm), 3, 0, 1, 0), BC5RgSnorm},
		{"bc6h_dx10", rawDX10(uint32(DxgiFormatBC6H_UF16), 3, 0, 1, 0), BC6hRgbUfloat},
		{"bc7_dx10", rawDX10(uint32(DxgiFormatBC7_UNorm), 3, 0, 1, 0), BC7RgbaUnorm},
		{"dxt1_legacy", rawLegacy(uint32(PixelFormatFlagsFOURCC), uint32(FourCCDXT1), 0, 0, 0, 0, 0), BC1RgbaUnorm},
		{"bc4u_fourcc", rawLegacy(uint32(PixelFormatFlagsFOURCC), uint32(FourCCBC4_UNORM), 0, 0, 0, 0, 0), BC4RUnorm},
		{"ati2_fourcc", rawLegacy(uint32(PixelFormatFlagsFOURCC), uint32(FourCCATI2), 0, 0, 0, 0, 0), BC5RgUnorm},
	}
	for _, c := range cases {
		dds, err := Parse(c.raw)
		if err != nil {
			t.Fatalf("%s: parse: %v", c.name, err)
		}
		format, err := DDSImageFormat(dds)
		if err != nil {
			t.Fatalf("%s: format: %v", c.name, err)
		}
		if format != c.format {
			t.Errorf("%s: got format %v want %v", c.name, format, c.format)
		}
	}
}

func TestSurfaceDdsRoundTrip(t *testing.T) {
	for _, format := range AllImageFormats() {
		data := make([]uint8, 4*4*6*format.BlockSizeInBytes())
		surface := &Surface{
			Width:       4,
			Height:      4,
			Depth:       1,
			Layers:      1,
			Mipmaps:     1,
			ImageFormat: format,
			Data:        data,
		}
		dds, err := surface.ToDds()
		if err != nil {
			t.Fatalf("ToDds %v: %v", format, err)
		}
		parsed, err := SurfaceFromDds(dds)
		if err != nil {
			t.Fatalf("SurfaceFromDds %v: %v", format, err)
		}
		if parsed.Width != surface.Width || parsed.Height != surface.Height ||
			parsed.Depth != surface.Depth || parsed.Layers != surface.Layers ||
			parsed.Mipmaps != surface.Mipmaps || parsed.ImageFormat != surface.ImageFormat {
			t.Errorf("%v: round trip mismatch: %+v", format, parsed)
		}
	}
}

func TestSurfaceDdsRoundTripCube(t *testing.T) {
	for _, format := range AllImageFormats() {
		data := make([]uint8, 4*4*6*format.BlockSizeInBytes())
		surface := &Surface{
			Width:       4,
			Height:      4,
			Depth:       1,
			Layers:      6,
			Mipmaps:     1,
			ImageFormat: format,
			Data:        data,
		}
		dds, err := surface.ToDds()
		if err != nil {
			t.Fatalf("ToDds %v: %v", format, err)
		}
		parsed, err := SurfaceFromDds(dds)
		if err != nil {
			t.Fatalf("SurfaceFromDds %v: %v", format, err)
		}
		if parsed.Layers != 6 {
			t.Errorf("%v: cube round trip layers = %d", format, parsed.Layers)
		}
	}
}

func TestSurfaceDdsRoundTrip3D(t *testing.T) {
	for _, format := range AllImageFormats() {
		data := make([]uint8, 4*4*4*format.BlockSizeInBytes())
		surface := &Surface{
			Width:       4,
			Height:      4,
			Depth:       4,
			Layers:      1,
			Mipmaps:     1,
			ImageFormat: format,
			Data:        data,
		}
		dds, err := surface.ToDds()
		if err != nil {
			t.Fatalf("ToDds %v: %v", format, err)
		}
		parsed, err := SurfaceFromDds(dds)
		if err != nil {
			t.Fatalf("SurfaceFromDds %v: %v", format, err)
		}
		if parsed.Depth != 4 {
			t.Errorf("%v: 3D round trip depth = %d", format, parsed.Depth)
		}
	}
}

func TestDdsBytesRoundTrip(t *testing.T) {
	// Serialize and re-parse must preserve the header fields.
	surface := &Surface{
		Width:       8,
		Height:      16,
		Depth:       1,
		Layers:      1,
		Mipmaps:     4,
		ImageFormat: BC7RgbaUnorm,
		Data:        make([]uint8, 8*16*16),
	}
	dds, err := surface.ToDds()
	if err != nil {
		t.Fatal(err)
	}
	serialized, err := dds.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := Parse(serialized)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.GetWidth() != 8 || parsed.GetHeight() != 16 || parsed.GetNumMipmapLevels() != 4 {
		t.Fatalf("round trip mismatch: %+v", parsed.Header)
	}
	if len(parsed.Data) != len(dds.Data) {
		t.Fatalf("data length mismatch: %d != %d", len(parsed.Data), len(dds.Data))
	}
}

func TestMaxMipmapCount(t *testing.T) {
	cases := []struct {
		in, want uint32
	}{
		{0, 0},
		{1, 1},
		{12, 4},
		{4, 3},
		{8, 4},
		{2, 2},
	}
	for _, c := range cases {
		if got := maxMipmapCount(c.in); got != c.want {
			t.Errorf("maxMipmapCount(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestCalculateOffset(t *testing.T) {
	cases := []struct {
		layer, depthLevel, mipmap uint32
		dimensions                [3]uint32
		blockDimensions           [3]uint32
		blockSize                 int
		mipmapsPerLayer           uint32
		want                      int
	}{
		{0, 0, 0, [3]uint32{8, 8, 8}, [3]uint32{4, 4, 4}, 16, 4, 0},
		{0, 0, 2, [3]uint32{8, 8, 8}, [3]uint32{4, 4, 4}, 16, 4, 128 + 16},
		{2, 0, 0, [3]uint32{8, 8, 8}, [3]uint32{4, 4, 4}, 16, 4, (128 + 16 + 16 + 16) * 2},
		{2, 0, 2, [3]uint32{8, 8, 8}, [3]uint32{4, 4, 4}, 16, 4, (128+16+16+16)*2 + 128 + 16},
		{0, 2, 0, [3]uint32{15, 15, 15}, [3]uint32{4, 4, 4}, 16, 1, 16 * 16 * 2},
		{0, 3, 0, [3]uint32{16, 16, 16}, [3]uint32{1, 1, 1}, 4, 1, 16 * 16 * 3 * 4},
	}
	for _, c := range cases {
		got, ok := calculateOffset(c.layer, c.depthLevel, c.mipmap, c.dimensions, c.blockDimensions, c.blockSize, c.mipmapsPerLayer)
		if !ok {
			t.Errorf("calculateOffset(%+v) failed", c)
			continue
		}
		if got != c.want {
			t.Errorf("calculateOffset(%+v) = %d, want %d", c, got, c.want)
		}
	}
}

func TestDownsampleRgba8(t *testing.T) {
	// Test that a checkerboard is averaged.
	original := make([]uint8, 4*4*4)
	for i := 0; i < 4*4; i += 2 {
		original[i*4] = 0
		original[i*4+1] = 0
		original[i*4+2] = 0
		original[i*4+3] = 0
		original[(i+1)*4] = 255
		original[(i+1)*4+1] = 255
		original[(i+1)*4+2] = 255
		original[(i+1)*4+3] = 255
	}
	got := downsampleRgba(2, 2, 1, 4, 4, 1, original)
	want := make([]uint8, 2*2*4)
	for i := range want {
		want[i] = 127
	}
	if !equalBytes(got, want) {
		t.Errorf("downsample 4x4 = %v want %v", got, want)
	}
}

func TestDownsampleRgba8_3x3(t *testing.T) {
	original := make([]uint8, 3*3*4)
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			idx := (i*3 + j) * 4
			if (i*3+j)%3 == 0 {
				original[idx] = 0
				original[idx+1] = 0
				original[idx+2] = 0
				original[idx+3] = 0
			} else {
				original[idx] = 255
				original[idx+1] = 255
				original[idx+2] = 255
				original[idx+3] = 255
			}
		}
	}
	got := downsampleRgba(1, 1, 1, 3, 3, 1, original)
	want := []uint8{127, 127, 127, 127}
	if !equalBytes(got, want) {
		t.Errorf("downsample 3x3 = %v want %v", got, want)
	}
}

func TestDecodeAllFormats(t *testing.T) {
	// Decoding must succeed for every format with a single 4x4 block.
	for _, format := range AllImageFormats() {
		data := make([]uint8, 4*4*format.BlockSizeInBytes())
		surface := &Surface{
			Width:       4,
			Height:      4,
			Depth:       1,
			Layers:      1,
			Mipmaps:     1,
			ImageFormat: format,
			Data:        data,
		}
		if _, err := surface.DecodeRgba8(); err != nil {
			t.Errorf("decode %v: %v", format, err)
		}
		if _, err := surface.DecodeRgbaf32(); err != nil {
			t.Errorf("decode f32 %v: %v", format, err)
		}
	}
}

func equalBytes(a, b []uint8) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
