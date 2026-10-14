package ddsutil

import (
	"math/rand"
	"testing"
)

// TestBcEncodeDecodeSolid verifies that encoding a solid color block and
// decoding it reproduces the color within the format's precision.
func TestBcEncodeDecodeSolid(t *testing.T) {
	cases := []struct {
		name   string
		bc     bcFormat
		pixel  [4]uint8
		maxErr int
	}{
		{"bc1", bcFmt1, [4]uint8{128, 64, 32, 255}, 8},
		{"bc2", bcFmt2, [4]uint8{200, 100, 50, 128}, 16},
		{"bc3", bcFmt3, [4]uint8{200, 100, 50, 128}, 8},
		{"bc4", bcFmt4, [4]uint8{100, 0, 0, 255}, 2},
		{"bc5", bcFmt5, [4]uint8{100, 200, 0, 255}, 2},
		{"bc7", bcFmt7, [4]uint8{200, 100, 50, 128}, 4},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			var block [16][4]uint8
			for i := 0; i < 16; i++ {
				block[i] = c.pixel
			}
			var compressed []byte
			switch c.bc {
			case bcFmt1:
				c := compressBc1Block(block, QualityNormal)
				compressed = c[:]
			case bcFmt2:
				c := compressBc2Block(block, QualityNormal)
				compressed = c[:]
			case bcFmt3:
				c := compressBc3Block(block, QualityNormal)
				compressed = c[:]
			case bcFmt4:
				var values [16]uint8
				for i := 0; i < 16; i++ {
					values[i] = block[i][0]
				}
				c := compressBc4Block(values, false, QualityNormal)
				compressed = c[:]
			case bcFmt5:
				var r, g [16]uint8
				for i := 0; i < 16; i++ {
					r[i] = block[i][0]
					g[i] = block[i][1]
				}
				c := compressBc5Block(r, g, false, QualityNormal)
				compressed = c[:]
			case bcFmt7:
				c := compressBc7Block(block, QualityNormal)
				compressed = c[:]
			}
			decoded := decompressBlock(c.bc, compressed)
			for y := 0; y < 4; y++ {
				for x := 0; x < 4; x++ {
					p := decoded[y][x]
					// Check the channels this format stores:
					// BC4 decodes to grayscale, BC5 to RG, the rest RGBA.
					channelsToCheck := 4
					if c.bc == bcFmt4 {
						channelsToCheck = 1
					} else if c.bc == bcFmt5 {
						channelsToCheck = 2
					}
					for ch := 0; ch < channelsToCheck; ch++ {
						d := int(p[ch]) - int(c.pixel[ch])
						if d < 0 {
							d = -d
						}
						if d > c.maxErr {
							t.Fatalf("pixel (%d,%d) channel %d: got %d want %d (err %d > %d)",
								x, y, ch, p[ch], c.pixel[ch], d, c.maxErr)
						}
					}
				}
			}
		})
	}
}

// TestBc6hEncodeDecode verifies the BC6H encoder through a decode round trip.
func TestBc6hEncodeDecode(t *testing.T) {
	cases := []struct {
		name  string
		value float32
	}{
		{"zero", 0.0},
		{"one", 1.0},
		{"half", 0.5},
		{"quarter", 0.25},
		{"small", 0.1},
		{"large", 100.0},
		{"two", 2.0},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			var block [16][4]uint16
			h := float32ToHalf(c.value)
			for i := 0; i < 16; i++ {
				block[i] = [4]uint16{h, h, h, 0x3C00}
			}
			compressed := compressBc6Block(block, QualityNormal)
			decoded := decompressBlockF32(bcFmt6, compressed[:])
			for y := 0; y < 4; y++ {
				for x := 0; x < 4; x++ {
					p := decoded[y][x]
					for ch := 0; ch < 3; ch++ {
						d := float64(p[ch]) - float64(c.value)
						if d < 0 {
							d = -d
						}
						// BC6H mode 11 stores 10-bit endpoints; the relative
						// error is around 1/31 of the half value.
						if d > float64(c.value)/30.0+0.001 {
							t.Fatalf("pixel (%d,%d) channel %d: got %v want %v", x, y, ch, p[ch], c.value)
						}
					}
				}
			}
		})
	}
}

// TestBc7EncodeDecodeGradient verifies BC7 with a gradient block (all 16
// pixels distinct), which exercises the 4-bit index assignment.
func TestBc7EncodeDecodeGradient(t *testing.T) {
	var block [16][4]uint8
	for i := 0; i < 16; i++ {
		v := uint8(i * 16)
		block[i] = [4]uint8{v, uint8(255 - v), v / 2, 255}
	}
	compressed := compressBc7Block(block, QualityNormal)
	decoded := decompressBlock(bcFmt7, compressed[:])
	for i := 0; i < 16; i++ {
		p := decoded[i/4][i%4]
		for ch := 0; ch < 3; ch++ {
			d := int(p[ch]) - int(block[i][ch])
			if d < 0 {
				d = -d
			}
			if d > 16 {
				t.Fatalf("pixel %d channel %d: got %d want %d", i, ch, p[ch], block[i][ch])
			}
		}
	}
}

// TestBc1TransparentMode verifies the BC1 3-color mode with transparency.
func TestBc1TransparentMode(t *testing.T) {
	var block [16][4]uint8
	for i := 0; i < 16; i++ {
		// Alternate opaque and transparent pixels.
		if i%2 == 0 {
			block[i] = [4]uint8{255, 0, 0, 255}
		} else {
			block[i] = [4]uint8{10, 10, 10, 0}
		}
	}
	compressed := compressBc1Block(block, QualityNormal)
	decoded := decompressBlock(bcFmt1, compressed[:])
	for i := 0; i < 16; i++ {
		p := decoded[i/4][i%4]
		if i%2 == 0 {
			if p[3] != 255 || p[0] < 200 {
				t.Fatalf("opaque pixel %d: got %v", i, p)
			}
		} else {
			if p[3] != 0 {
				t.Fatalf("transparent pixel %d: got %v", i, p)
			}
		}
	}
}

// TestBc3Alpha verifies BC3 alpha endpoint fitting.
func TestBc3Alpha(t *testing.T) {
	var block [16][4]uint8
	for i := 0; i < 16; i++ {
		block[i] = [4]uint8{0, 0, 0, uint8(i * 16)}
	}
	compressed := compressBc3Block(block, QualityNormal)
	decoded := decompressBlock(bcFmt3, compressed[:])
	for i := 0; i < 16; i++ {
		p := decoded[i/4][i%4]
		d := int(p[3]) - int(block[i][3])
		if d < 0 {
			d = -d
		}
		// 8 palette levels cover 16 distinct values, so the per-pixel error
		// is bounded by roughly half the palette step.
		if d > 20 {
			t.Fatalf("alpha pixel %d: got %d want %d", i, p[3], block[i][3])
		}
	}
}

// TestEncodeDecodeDdsRoundTrip encodes an RGBA8 surface to each BC format,
// serializes to DDS, decodes, and checks the result is within tolerance.
// The pattern is a 1D ramp along x so the colors lie on a line in RGB space,
// which block compressed formats can represent well.
func TestEncodeDecodeDdsRoundTrip(t *testing.T) {
	formats := []struct {
		format          ImageFormat
		maxErr          int
		channelsToCheck int
	}{
		{BC1RgbaUnorm, 10, 4},
		{BC2RgbaUnorm, 10, 4},
		{BC3RgbaUnorm, 10, 4},
		{BC4RUnorm, 8, 1},
		{BC4RSnorm, 8, 1},
		{BC5RgUnorm, 8, 2},
		{BC5RgSnorm, 8, 2},
		{BC6hRgbUfloat, 10, 3},
		{BC7RgbaUnorm, 4, 4},
	}
	for _, tc := range formats {
		tc := tc
		t.Run(tc.format.String(), func(t *testing.T) {
			// Build a 8x8 RGBA surface with a 1D gradient along x
			// (repeating every 4 columns so each 4x4 block is identical).
			// Values stay in [64, 255] so BC6H's half-space interpolation
			// (logarithmic in float space) stays reasonably dense.
			data := make([]uint8, 8*8*4)
			for y := 0; y < 8; y++ {
				for x := 0; x < 8; x++ {
					idx := (y*8 + x) * 4
					data[idx] = uint8(128 + (x%4)*32)
					data[idx+1] = uint8(255 - (x%4)*32)
					data[idx+2] = uint8(64 + (x%4)*16)
					data[idx+3] = 255
				}
			}
			surface := &SurfaceRgba8{Width: 8, Height: 8, Depth: 1, Layers: 1, Mipmaps: 1, Data: data}
			encoded, err := surface.Encode(tc.format, QualityNormal, MipmapsDisabled)
			if err != nil {
				t.Fatal(err)
			}
			dds, err := encoded.ToDds()
			if err != nil {
				t.Fatal(err)
			}
			bytes, err := dds.Bytes()
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := Parse(bytes)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := ImageFromDds(parsed, 0)
			if err != nil {
				t.Fatal(err)
			}
			if decoded.Rect.Dx() != 8 || decoded.Rect.Dy() != 8 {
				t.Fatalf("decoded dimensions %v", decoded.Rect)
			}
			// Check the decoded image is a reasonable approximation.
			var maxErr int
			for i := 0; i < len(data); i += 4 {
				for ch := 0; ch < tc.channelsToCheck; ch++ {
					d := int(decoded.Pix[i+ch]) - int(data[i+ch])
					if d < 0 {
						d = -d
					}
					if d > maxErr {
						maxErr = d
					}
				}
			}
			if maxErr > tc.maxErr {
				t.Fatalf("max channel error %d exceeds %d", maxErr, tc.maxErr)
			}
		})
	}
}

// TestSnormEncodeDecodeF32 checks signed BC4/BC5 data, including negative
// values, survives an RGBAF32 round trip.
func TestSnormEncodeDecodeF32(t *testing.T) {
	for _, format := range []ImageFormat{BC4RSnorm, BC5RgSnorm} {
		format := format
		t.Run(format.String(), func(t *testing.T) {
			data := make([]float32, 8*8*4)
			for y := 0; y < 8; y++ {
				for x := 0; x < 8; x++ {
					idx := (y*8 + x) * 4
					data[idx] = -1.0 + float32(x)*0.25
					data[idx+1] = 0.75 - float32(y)*0.25
					data[idx+3] = 1.0
				}
			}
			surface := &SurfaceRgba32Float{Width: 8, Height: 8, Depth: 1, Layers: 1, Mipmaps: 1, Data: data}
			encoded, err := surface.Encode(format, QualityNormal, MipmapsDisabled)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := encoded.DecodeRgbaf32()
			if err != nil {
				t.Fatal(err)
			}
			channels := 1
			if format == BC5RgSnorm {
				channels = 2
			}
			for i := 0; i < len(data); i += 4 {
				for c := 0; c < channels; c++ {
					d := decoded.Data[i+c] - data[i+c]
					if d < -0.05 || d > 0.05 {
						t.Fatalf("pixel %d channel %d: got %v want %v", i/4, c, decoded.Data[i+c], data[i+c])
					}
				}
			}
		})
	}
}

// TestFixUpIndexMostSignificantBit checks blocks whose first pixel is best
// represented by the second endpoint. The fix-up index of BC6H and BC7 is
// stored without its most significant bit, so the encoder must swap the
// endpoints instead of truncating the index.
func TestFixUpIndexMostSignificantBit(t *testing.T) {
	for name, bc := range map[string]bcFormat{"BC6H": bcFmt6, "BC7": bcFmt7} {
		var block [16][4]uint8
		for i := range block {
			block[i] = [4]uint8{0, 0, 0, 255}
		}
		// Only the first pixel is bright, so the encoder picks an index of
		// the bright endpoint for it.
		block[0] = [4]uint8{200, 160, 120, 255}

		var compressed []byte
		if bc == bcFmt7 {
			c := compressBc7Block(block, QualityNormal)
			compressed = c[:]
		} else {
			var half [16][4]uint16
			for i := range block {
				for c := 0; c < 4; c++ {
					half[i][c] = float32ToHalf(float32(block[i][c]) / 255.0)
				}
			}
			c := compressBc6Block(half, QualityNormal)
			compressed = c[:]
		}
		decoded := decompressBlock(bc, compressed)
		for c := 0; c < 3; c++ {
			d := int(decoded[0][0][c]) - int(block[0][c])
			if d < -12 || d > 12 {
				t.Errorf("%s: first pixel %v, want %v", name, decoded[0][0], block[0])
				break
			}
		}
	}
}

// TestBc1LayoutMatchesDecoder checks that the error evaluateBc1 reports for a
// layout equals the error of the block the decoder reconstructs, for both
// endpoint orders. BC2/BC3 color blocks always decode with the 4-color
// palette, BC1 blocks pick it only when c0 > c1.
func TestBc1LayoutMatchesDecoder(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	for iteration := 0; iteration < 200; iteration++ {
		var block [16][4]uint8
		for i := range block {
			block[i] = [4]uint8{uint8(r.Intn(256)), uint8(r.Intn(256)), uint8(r.Intn(256)), 255}
		}
		c0 := rgb565{uint8(r.Intn(32)), uint8(r.Intn(64)), uint8(r.Intn(32))}
		c1 := rgb565{uint8(r.Intn(32)), uint8(r.Intn(64)), uint8(r.Intn(32))}
		for _, tc := range []struct {
			layout bc1Layout
			bc     bcFormat
		}{{bc1Opaque, bcFmt1}, {bc1FourColor, bcFmt3}} {
			for _, order := range [][2]rgb565{{c0, c1}, {c1, c0}} {
				fit, ok := evaluateBc1(&block, order[0], order[1], tc.layout)
				if !ok {
					t.Fatal("opaque layouts accept every endpoint order")
				}
				var packed uint64
				packed |= uint64(pack565(fit.c0)) | uint64(pack565(fit.c1))<<16
				for i := 0; i < 16; i++ {
					packed |= uint64(fit.indices[i]&3) << uint(32+i*2)
				}
				var compressed [16]byte
				colorBlock := compressed[:8]
				if tc.bc == bcFmt3 {
					colorBlock = compressed[8:]
				}
				for i := 0; i < 8; i++ {
					colorBlock[i] = byte(packed >> uint(i*8))
				}
				decoded := decompressBlock(tc.bc, compressed[:])
				var err uint64
				for i := 0; i < 16; i++ {
					p := decoded[i/4][i%4]
					err += uint64(colorDistSq([3]uint8{p[0], p[1], p[2]}, [3]uint8{block[i][0], block[i][1], block[i][2]}))
				}
				if err != fit.err {
					t.Fatalf("layout %d: evaluated error %d, decoded error %d", tc.layout, fit.err, err)
				}
			}
		}
	}
}
