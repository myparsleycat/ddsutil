package ddsutil

// This file implements BCn block compressors for encoding.
//
// The encoders are self-contained Go implementations; porting Intel's ISPC
// texture compressor was not practical. They produce valid, reasonable
// quality output tuned independently of the ISPC reference.
//
// Encoder strategy per format:
//   - bcFmt1: exhaustive endpoint search over the block's quantized colors.
//   - bcFmt2: 4-bit alpha + bcFmt1-style 4-color block.
//   - bcFmt3: 8-bit alpha endpoints with exact bcdec interpolation + bcFmt1 color.
//   - bcFmt4/bcFmt5: min/max endpoints with exact bcdec interpolation.
//   - BC6H: mode 11 (10.10.10 endpoints, no delta compression).
//   - bcFmt7: mode 6 (7-bit RGBA endpoints, p-bits, 4-bit indices).

// encodeBcn encodes pixel data to a compressed BCn format.
func encodeBcn[T channel](bc bcFormat, width, height uint32, data []T, quality Quality) ([]byte, error) {
	// Surface dimensions are not validated yet and may cause overflow.
	expectedSize, ok := mipSize(int(width), int(height), 1, blockWidth, blockHeight, 1, elementsPerBlock)
	if !ok {
		return nil, &SurfaceError{Kind: SurfaceErrorPixelCountWouldOverflow, Width: width, Height: height, Depth: 1}
	}

	// The surface must be a multiple of the block dimensions for safety.
	if len(data) < expectedSize {
		return nil, &SurfaceError{Kind: SurfaceErrorNotEnoughData, Expected: expectedSize, Actual: len(data)}
	}

	blockSize := bc.blockSizeInBytes()
	out := make([]byte, 0, int(width/4)*int(height/4)*blockSize)

	switch bc {
	case bcFmt1, bcFmt2, bcFmt3, bcFmt4, bcFmt4S, bcFmt5, bcFmt5S, bcFmt7:
		rgba8, err := channelBytes(data)
		if err != nil {
			return nil, err
		}
		for y := uint32(0); y < height; y += blockHeight {
			for x := uint32(0); x < width; x += blockWidth {
				var block [16][4]uint8
				for j := uint32(0); j < 4; j++ {
					for i := uint32(0); i < 4; i++ {
						idx := ((y+j)*width + x + i) * 4
						p := rgba8[idx : idx+4]
						block[j*4+i] = [4]uint8{p[0], p[1], p[2], p[3]}
					}
				}
				var compressed []byte
				switch bc {
				case bcFmt1:
					compressed = compressBc1Block(block)
				case bcFmt2:
					compressed = compressBc2Block(block)
				case bcFmt3:
					compressed = compressBc3Block(block)
				case bcFmt4, bcFmt4S:
					var values [16]uint8
					for i := 0; i < 16; i++ {
						values[i] = block[i][0]
					}
					compressed = compressBc4Block(values)
				case bcFmt5, bcFmt5S:
					var r, g [16]uint8
					for i := 0; i < 16; i++ {
						r[i] = block[i][0]
						g[i] = block[i][1]
					}
					compressed = compressBc5Block(r, g)
				case bcFmt7:
					compressed = compressBc7Block(block)
				}
				out = append(out, compressed...)
			}
		}
	case bcFmt6:
		// The BC6H encoder expects the data in half precision floating point.
		var halfData []uint16
		switch any(*new(T)).(type) {
		case float32:
			floats := any(data).([]float32)
			halfData = make([]uint16, len(floats))
			for i, f := range floats {
				halfData[i] = float32ToHalf(f)
			}
		default:
			bytes := any(data).([]uint8)
			halfData = make([]uint16, len(bytes))
			for i, b := range bytes {
				halfData[i] = float32ToHalf(float32(b) / 255.0)
			}
		}
		for y := uint32(0); y < height; y += blockHeight {
			for x := uint32(0); x < width; x += blockWidth {
				var block [16][4]uint16
				for j := uint32(0); j < 4; j++ {
					for i := uint32(0); i < 4; i++ {
						idx := ((y+j)*width + x + i) * 4
						block[j*4+i] = [4]uint16{halfData[idx], halfData[idx+1], halfData[idx+2], halfData[idx+3]}
					}
				}
				out = append(out, compressBc6Block(block)...)
			}
		}
	}

	return out, nil
}

// channelBytes converts generic channel data to bytes (used for the u8-based
// BCn encoders).
func channelBytes[T channel](data []T) ([]uint8, error) {
	switch any(*new(T)).(type) {
	case uint8:
		return any(data).([]uint8), nil
	default:
		return nil, &SurfaceError{Kind: SurfaceErrorUnsupportedEncodeFormat}
	}
}

// ---------------------------------------------------------------------------
// bcFmt1
// ---------------------------------------------------------------------------

type rgb565 struct {
	r uint8
	g uint8
	b uint8
}

func pack565(c rgb565) uint16 {
	return uint16(c.r)<<11 | uint16(c.g)<<5 | uint16(c.b)
}

// expand565 matches bcdec's color_block expansion.
func expand565(c rgb565) [3]uint8 {
	r := (uint16(c.r)*527 + 23) >> 6
	g := (uint16(c.g)*259 + 33) >> 6
	b := (uint16(c.b)*527 + 23) >> 6
	return [3]uint8{uint8(r), uint8(g), uint8(b)}
}

// interpolateColor4 computes the 4-color palette for bcFmt1 (c0 > c1).
// Matches bcdec's color_block (u32 arithmetic).
func interpolateColor4(c0, c1 rgb565) [4][3]uint8 {
	var palette [4][3]uint8
	r0, g0, b0 := uint32(c0.r), uint32(c0.g), uint32(c0.b)
	r1, g1, b1 := uint32(c1.r), uint32(c1.g), uint32(c1.b)
	palette[0] = expand565(c0)
	palette[1] = expand565(c1)
	palette[2] = [3]uint8{
		uint8(((2*r0+r1)*351 + 61) >> 7),
		uint8(((2*g0+g1)*2763 + 1039) >> 11),
		uint8(((2*b0+b1)*351 + 61) >> 7),
	}
	palette[3] = [3]uint8{
		uint8(((r0+2*r1)*351 + 61) >> 7),
		uint8(((g0+2*g1)*2763 + 1039) >> 11),
		uint8(((b0+2*b1)*351 + 61) >> 7),
	}
	return palette
}

// interpolateColor3 computes the 3-color palette for bcFmt1 (c0 <= c1).
func interpolateColor3(c0, c1 rgb565) [4][3]uint8 {
	var palette [4][3]uint8
	r0, g0, b0 := uint32(c0.r), uint32(c0.g), uint32(c0.b)
	r1, g1, b1 := uint32(c1.r), uint32(c1.g), uint32(c1.b)
	palette[0] = expand565(c0)
	palette[1] = expand565(c1)
	palette[2] = [3]uint8{
		uint8(((r0+r1)*1053 + 125) >> 8),
		uint8(((g0+g1)*4145 + 1019) >> 11),
		uint8(((b0+b1)*1053 + 125) >> 8),
	}
	palette[3] = [3]uint8{0, 0, 0}
	return palette
}

func colorDistSq(a, b [3]uint8) uint32 {
	dr := int32(a[0]) - int32(b[0])
	dg := int32(a[1]) - int32(b[1])
	db := int32(a[2]) - int32(b[2])
	return uint32(dr*dr + dg*dg + db*db)
}

// fitBc1Color finds the best (c0, c1) for the block and writes the color
// indices into indices. hasTransparent indicates the block has pixels with
// alpha < 128, which requires the 3-color + transparent layout (c0 <= c1).
// Returns the packed color block (first 8 bytes are indices + colors).
func fitBc1Color(block [16][4]uint8, hasTransparent bool) (colorBlock uint64, indices [16]uint8) {
	// Gather the unique quantized colors in the block.
	var unique []rgb565
	var seen [65536]bool
	var all [16]rgb565
	for i := 0; i < 16; i++ {
		c := rgb565{block[i][0] >> 3, block[i][1] >> 2, block[i][2] >> 3}
		all[i] = c
		if !seen[pack565(c)] {
			seen[pack565(c)] = true
			unique = append(unique, c)
		}
	}
	if len(unique) == 0 {
		unique = append(unique, rgb565{})
	}

	bestError := ^uint64(0)
	var bestIndices [16]uint8
	var bestC0, bestC1 rgb565

	for _, c0 := range unique {
		for _, c1 := range unique {
			p0, p1 := pack565(c0), pack565(c1)

			// The decoder uses the 4-color layout when c0 > c1 (or always for
			// BC2/BC3 color blocks) and the 3-color + transparent layout
			// otherwise.
			useFourColor := !hasTransparent && p0 > p1
			if hasTransparent && p0 > p1 {
				// Transparent mode requires c0 <= c1.
				continue
			}

			var palette [4][3]uint8
			if useFourColor {
				palette = interpolateColor4(c0, c1)
			} else {
				palette = interpolateColor3(c0, c1)
			}

			var err uint64
			var idx [16]uint8
			for i := 0; i < 16; i++ {
				transparent := hasTransparent && block[i][3] < 128
				if transparent {
					// Transparent pixels must use index 3.
					idx[i] = 3
					continue
				}
				best := 0
				bestD := colorDistSq(palette[0], [3]uint8{block[i][0], block[i][1], block[i][2]})
				maxK := 3
				if !useFourColor {
					// Index 3 is transparent black in the 3-color layout.
					maxK = 2
				}
				for k := 1; k <= maxK; k++ {
					d := colorDistSq(palette[k], [3]uint8{block[i][0], block[i][1], block[i][2]})
					if d < bestD {
						bestD = d
						best = k
					}
				}
				idx[i] = uint8(best)
				err += uint64(bestD)
			}
			if err < bestError {
				bestError = err
				bestIndices = idx
				bestC0, bestC1 = c0, c1
			}
		}
	}

	// Pack: colors first (4 bytes), then 2-bit indices (4 bytes).
	var packed uint64
	packed |= uint64(pack565(bestC0))
	packed |= uint64(pack565(bestC1)) << 16
	for i := 0; i < 16; i++ {
		packed |= uint64(bestIndices[i]&3) << uint(32+i*2)
	}
	return packed, bestIndices
}

func compressBc1Block(block [16][4]uint8) []byte {
	hasTransparent := false
	for i := 0; i < 16; i++ {
		if block[i][3] < 128 {
			hasTransparent = true
			break
		}
	}
	colorBlock, _ := fitBc1Color(block, hasTransparent)
	out := make([]byte, 8)
	for i := 0; i < 8; i++ {
		out[i] = byte(colorBlock >> uint(i*8))
	}
	return out
}

// ---------------------------------------------------------------------------
// bcFmt2
// ---------------------------------------------------------------------------

func compressBc2Block(block [16][4]uint8) []byte {
	// Alpha: 4 bits per pixel.
	var alpha uint64
	for i := 0; i < 16; i++ {
		alpha |= uint64(block[i][3]/17) << uint(i*4)
	}
	colorBlock, _ := fitBc1Color(block, false)
	out := make([]byte, 16)
	for i := 0; i < 8; i++ {
		out[i] = byte(alpha >> uint(i*8))
		out[i+8] = byte(colorBlock >> uint(i*8))
	}
	return out
}

// ---------------------------------------------------------------------------
// bcFmt3
// ---------------------------------------------------------------------------

// interpolateAlpha6 matches bcdec's smooth_alpha_block 6-level interpolation.
func interpolateAlpha6(a0, a1 uint32) [8]uint32 {
	var alpha [8]uint32
	alpha[0] = a0
	alpha[1] = a1
	alpha[2] = (6*a0 + a1 + 1) / 7
	alpha[3] = (5*a0 + 2*a1 + 1) / 7
	alpha[4] = (4*a0 + 3*a1 + 1) / 7
	alpha[5] = (3*a0 + 4*a1 + 1) / 7
	alpha[6] = (2*a0 + 5*a1 + 1) / 7
	alpha[7] = (a0 + 6*a1 + 1) / 7
	return alpha
}

// interpolateAlpha4 matches bcdec's smooth_alpha_block 4-level interpolation.
func interpolateAlpha4(a0, a1 uint32) [8]uint32 {
	var alpha [8]uint32
	alpha[0] = a0
	alpha[1] = a1
	alpha[2] = (4*a0 + a1 + 1) / 5
	alpha[3] = (3*a0 + 2*a1 + 1) / 5
	alpha[4] = (2*a0 + 3*a1 + 1) / 5
	alpha[5] = (a0 + 4*a1 + 1) / 5
	alpha[6] = 0
	alpha[7] = 255
	return alpha
}

func compressSmoothAlpha(alphas [16]uint8) [8]byte {
	// Collect the distinct alpha values as endpoint candidates.
	var unique []uint32
	var seen [256]bool
	for i := 0; i < 16; i++ {
		if !seen[alphas[i]] {
			seen[alphas[i]] = true
			unique = append(unique, uint32(alphas[i]))
		}
	}

	bestError := uint64(0xFFFFFFFFFFFFFFFF)
	var bestE0, bestE1 uint32
	var bestIndices uint64

	// Search endpoint pairs (a0 > a1 uses the 6-level layout; a0 <= a1 uses
	// the 4-level layout with fixed 0/255 entries).
	for _, a0 := range unique {
		for _, a1 := range unique {
			var palette [8]uint32
			if a0 > a1 {
				palette = interpolateAlpha6(a0, a1)
			} else {
				palette = interpolateAlpha4(a0, a1)
			}

			var err uint64
			var indices uint64
			for i := 0; i < 16; i++ {
				best := 0
				bestD := absDiffU32(palette[0], uint32(alphas[i]))
				for k := 1; k < 8; k++ {
					d := absDiffU32(palette[k], uint32(alphas[i]))
					if d < bestD {
						bestD = d
						best = k
					}
				}
				indices |= uint64(best) << uint(i*3)
				err += uint64(bestD)
			}
			if err < bestError {
				bestError = err
				bestE0, bestE1 = a0, a1
				bestIndices = indices
			}
		}
	}

	var out [8]byte
	out[0] = byte(bestE0)
	out[1] = byte(bestE1)
	for i := 0; i < 6; i++ {
		out[2+i] = byte(bestIndices >> uint(i*8))
	}
	return out
}

func compressBc3Block(block [16][4]uint8) []byte {
	var alphas [16]uint8
	for i := 0; i < 16; i++ {
		alphas[i] = block[i][3]
	}
	alphaBlock := compressSmoothAlpha(alphas)
	colorBlock, _ := fitBc1Color(block, false)
	out := make([]byte, 16)
	for i := 0; i < 8; i++ {
		out[i] = alphaBlock[i]
		out[i+8] = byte(colorBlock >> uint(i*8))
	}
	return out
}

// ---------------------------------------------------------------------------
// bcFmt4 / bcFmt5
// ---------------------------------------------------------------------------

// interpolateBc4 matches bcdec's bc4_block interpolation.
func interpolateBc4(a0, a1 uint32, isSigned bool) [8]int32 {
	var aWeights4 = [4]int32{13107, 26215, 39321, 52429}
	var aWeights6 = [6]int32{9363, 18724, 28086, 37450, 46812, 56173}

	var alpha [8]int32
	alpha[0] = int32(a0)
	alpha[1] = int32(a1)
	if alpha[0] > alpha[1] {
		alpha[2] = (aWeights6[5]*alpha[0] + aWeights6[0]*alpha[1] + 32768) >> 16
		alpha[3] = (aWeights6[4]*alpha[0] + aWeights6[1]*alpha[1] + 32768) >> 16
		alpha[4] = (aWeights6[3]*alpha[0] + aWeights6[2]*alpha[1] + 32768) >> 16
		alpha[5] = (aWeights6[2]*alpha[0] + aWeights6[3]*alpha[1] + 32768) >> 16
		alpha[6] = (aWeights6[1]*alpha[0] + aWeights6[4]*alpha[1] + 32768) >> 16
		alpha[7] = (aWeights6[0]*alpha[0] + aWeights6[5]*alpha[1] + 32768) >> 16
	} else {
		alpha[2] = (aWeights4[3]*alpha[0] + aWeights4[0]*alpha[1] + 32768) >> 16
		alpha[3] = (aWeights4[2]*alpha[0] + aWeights4[1]*alpha[1] + 32768) >> 16
		alpha[4] = (aWeights4[1]*alpha[0] + aWeights4[2]*alpha[1] + 32768) >> 16
		alpha[5] = (aWeights4[0]*alpha[0] + aWeights4[3]*alpha[1] + 32768) >> 16
		if isSigned {
			alpha[6] = -127
		} else {
			alpha[6] = 0
		}
		if isSigned {
			alpha[7] = 127
		} else {
			alpha[7] = 255
		}
	}
	return alpha
}

func compressBc4Block(values [16]uint8) []byte {
	// Collect the distinct values as endpoint candidates.
	var unique []uint32
	var seen [256]bool
	for i := 0; i < 16; i++ {
		if !seen[values[i]] {
			seen[values[i]] = true
			unique = append(unique, uint32(values[i]))
		}
	}

	bestError := uint64(0xFFFFFFFFFFFFFFFF)
	var bestE0, bestE1 uint32
	var bestIndices uint64

	for _, a0 := range unique {
		for _, a1 := range unique {
			palette := interpolateBc4(a0, a1, false)

			var err uint64
			var indices uint64
			for i := 0; i < 16; i++ {
				best := 0
				bestD := absI32(palette[0] - int32(values[i]))
				for k := 1; k < 8; k++ {
					d := absI32(palette[k] - int32(values[i]))
					if d < bestD {
						bestD = d
						best = k
					}
				}
				indices |= uint64(best) << uint(i*3)
				err += uint64(bestD)
			}
			if err < bestError {
				bestError = err
				bestE0, bestE1 = a0, a1
				bestIndices = indices
			}
		}
	}

	var out [8]byte
	out[0] = byte(bestE0)
	out[1] = byte(bestE1)
	for i := 0; i < 6; i++ {
		out[2+i] = byte(bestIndices >> uint(i*8))
	}
	return out[:]
}

func absI32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func compressBc5Block(r, g [16]uint8) []byte {
	rBlock := compressBc4Block(r)
	gBlock := compressBc4Block(g)
	out := make([]byte, 16)
	copy(out[:8], rBlock)
	copy(out[8:], gBlock)
	return out
}

// ---------------------------------------------------------------------------
// BC6H
// ---------------------------------------------------------------------------

// compressBc6Block encodes a block using BC6H mode 11 (0b00011):
// no partition, both endpoints stored explicitly with 10 bits each.
func compressBc6Block(block [16][4]uint16) []byte {
	// For unsigned mode, clamp negative halves (sign bit set) to zero.
	var minV [3]uint16
	var maxV [3]uint16
	for c := 0; c < 3; c++ {
		minV[c] = 0xFFFF
		maxV[c] = 0
	}
	for i := 0; i < 16; i++ {
		for c := 0; c < 3; c++ {
			v := block[i][c] & 0x7FFF
			if v < minV[c] {
				minV[c] = v
			}
			if v > maxV[c] {
				maxV[c] = v
			}
		}
	}

	// Decoded half bits for a 10-bit endpoint e are approximately e*31.
	// Invert: e = round(h / 31).
	// Try all 8 per-channel endpoint orientations (min/max assignment).
	var weights = [16]int32{0, 4, 9, 13, 17, 21, 26, 30, 34, 38, 43, 47, 51, 55, 60, 64}

	bestError := float64(0xFFFFFFFFFFFFFFFF)
	var bestE0, bestE1 [3]uint16
	var bestIndices uint64

	for orientation := 0; orientation < 8; orientation++ {
		var e0, e1 [3]uint16
		for c := 0; c < 3; c++ {
			flipped := (orientation>>c)&1 == 1
			if flipped {
				e0[c] = clampU16(uint16((uint32(maxV[c])+15)/31), 1023)
				e1[c] = clampU16(uint16((uint32(minV[c])+15)/31), 1023)
			} else {
				e0[c] = clampU16(uint16((uint32(minV[c])+15)/31), 1023)
				e1[c] = clampU16(uint16((uint32(maxV[c])+15)/31), 1023)
			}
		}

		// Assign indices: 4-bit weights, palette via bcdec interpolation.
		// The decoder interpolates the unquantized endpoint values
		// (e*64+32) and scales by 31/64, giving half bits
		// h = interp*31 + 15. Compare in float space.
		var err float64
		var indices uint64
		for i := 0; i < 16; i++ {
			best := 0
			bestErr := float64(0xFFFFFFFFFFFFFFFF)
			for k := 0; k < 16; k++ {
				var e float64
				for c := 0; c < 3; c++ {
					interp := interpolateI32(int32(e0[c]), int32(e1[c]), weights[k])
					h := (interp*1984 + 992) >> 6 // finish_unquantize unsigned
					decoded := halfToFloat32(uint16(h))
					target := halfToFloat32(block[i][c] & 0x7FFF)
					d := float64(decoded - target)
					e += d * d
				}
				if e < bestErr {
					bestErr = e
					best = k
				}
			}
			indices |= uint64(best) << uint(i*4)
			err += bestErr
		}
		if err < bestError {
			bestError = err
			bestE0, bestE1 = e0, e1
			bestIndices = indices
		}
	}

	// Assemble the bitstream LSB-first.
	// Mode 11: 0b00011 LSB-first: 1, 1, 0, 0, 0; endpoints; then indices
	// (first pixel uses 3 bits as the fix-up, the rest 4 bits).
	return assembleBc6Mode11(bestE0, bestE1, bestIndices)
}

// assembleBc6Mode11 builds the 128-bit mode 11 block.
func assembleBc6Mode11(e0, e1 [3]uint16, indices uint64) []byte {
	var words [16]byte
	// bit writer over the 16 bytes, LSB-first
	bitPos := uint(0)
	write := func(value uint64, count uint) {
		for i := uint(0); i < count; i++ {
			bit := (value >> i) & 1
			if bit == 1 {
				words[bitPos/8] |= 1 << (bitPos % 8)
			}
			bitPos++
		}
	}
	// Mode 11: 0b00011 LSB-first: bits 0,1 = 1,1; bits 2-4 = 0,0,0
	write(0b11, 2)
	write(0, 3)
	for c := 0; c < 3; c++ {
		write(uint64(e0[c]), 10)
	}
	for c := 0; c < 3; c++ {
		write(uint64(e1[c]), 10)
	}
	// Indices: the fix-up pixel (0) uses 3 bits, the rest use 4 bits.
	// The fix-up's 4th bit is dropped, so write the stream per pixel.
	for i := 0; i < 16; i++ {
		count := uint(4)
		if i == 0 {
			count = 3
		}
		write((indices>>uint(i*4))&((1<<count)-1), count)
	}
	return words[:]
}

func clampU16(v uint16, max uint16) uint16 {
	if v > max {
		return max
	}
	return v
}

// ---------------------------------------------------------------------------
// bcFmt7
// ---------------------------------------------------------------------------

// bc7Weight4 is the 4-bit index weight table.
var bc7Weight4 = [16]uint32{0, 4, 9, 13, 17, 21, 26, 30, 34, 38, 43, 47, 51, 55, 60, 64}

// compressBc7Block encodes a block using BC7 mode 6:
// 1 subset, 7-bit RGBA endpoints with unique p-bits, 4-bit indices.
func compressBc7Block(block [16][4]uint8) []byte {
	// Per-channel min/max of the block.
	var minV, maxV [4]uint8
	for c := 0; c < 4; c++ {
		minV[c] = 255
		maxV[c] = 0
	}
	for i := 0; i < 16; i++ {
		for c := 0; c < 4; c++ {
			if block[i][c] < minV[c] {
				minV[c] = block[i][c]
			}
			if block[i][c] > maxV[c] {
				maxV[c] = block[i][c]
			}
		}
	}

	// Endpoints are per-channel independent, so each channel can map its
	// min to either endpoint. Try all 16 orientation combinations.
	bestError := uint64(0xFFFFFFFFFFFFFFFF)
	var bestE0, bestE1 [4]uint8
	var bestP0, bestP1 uint8
	var bestIndices uint64

	for orientation := 0; orientation < 16; orientation++ {
		var e0, e1 [4]uint8
		var t0, t1 [4]uint8
		for c := 0; c < 4; c++ {
			flipped := (orientation>>c)&1 == 1
			if flipped {
				e0[c] = maxV[c] >> 1
				e1[c] = minV[c] >> 1
				t0[c] = maxV[c]
				t1[c] = minV[c]
			} else {
				e0[c] = minV[c] >> 1
				e1[c] = maxV[c] >> 1
				t0[c] = minV[c]
				t1[c] = maxV[c]
			}
		}
		// Endpoint components are 7 bits + 1 shared p-bit per endpoint,
		// decoded as (e7<<1)|p. Choose the p-bit that best fits the channel
		// values the endpoint was derived from.
		p0 := bestPBit(e0, t0)
		p1 := bestPBit(e1, t1)

		// Decoded endpoint values (8-bit).
		var d0, d1 [4]uint32
		for c := 0; c < 4; c++ {
			d0[c] = uint32(e0[c])<<1 | uint32(p0)
			d1[c] = uint32(e1[c])<<1 | uint32(p1)
		}

		// Assign 4-bit indices and accumulate the error.
		var err uint64
		var indices uint64
		for i := 0; i < 16; i++ {
			best := 0
			bestErr := uint64(0xFFFFFFFFFFFFFFFF)
			for k := 0; k < 16; k++ {
				w := bc7Weight4[k]
				var e uint64
				for c := 0; c < 4; c++ {
					decoded := (d0[c]*(64-w) + d1[c]*w + 32) >> 6
					e += uint64(absDiffU32(decoded, uint32(block[i][c])))
				}
				if e < bestErr {
					bestErr = e
					best = k
				}
			}
			indices |= uint64(best) << uint(i*4)
			err += bestErr
		}
		if err < bestError {
			bestError = err
			bestE0, bestE1 = e0, e1
			bestP0, bestP1 = p0, p1
			bestIndices = indices
		}
	}

	// Assemble the bitstream LSB-first.
	var words [16]byte
	bitPos := uint(0)
	write := func(value uint64, count uint) {
		for i := uint(0); i < count; i++ {
			if ((value >> i) & 1) == 1 {
				words[bitPos/8] |= 1 << (bitPos % 8)
			}
			bitPos++
		}
	}
	// Mode 6: six zeros then a 1.
	write(0, 6)
	write(1, 1)
	// Endpoints are stored channel-major (matching the decoder):
	// R0, R1, G0, G1, B0, B1, A0, A1.
	for c := 0; c < 4; c++ {
		write(uint64(bestE0[c]), 7)
		write(uint64(bestE1[c]), 7)
	}
	// P-bits.
	write(uint64(bestP0), 1)
	write(uint64(bestP1), 1)
	// Indices: the fix-up pixel (0) uses 3 bits, the rest use 4 bits.
	// The fix-up's 4th bit is dropped, so write the stream per pixel.
	for i := 0; i < 16; i++ {
		count := uint(4)
		if i == 0 {
			count = 3
		}
		write((bestIndices>>uint(i*4))&((1<<count)-1), count)
	}

	return words[:]
}

// bestPBit picks the p-bit that minimizes the total error of the quantized
// endpoint components relative to the target values.
func bestPBit(e, target [4]uint8) uint8 {
	var err0, err1 uint32
	for c := 0; c < 4; c++ {
		d0 := uint32(e[c])<<1 | 0
		d1 := uint32(e[c])<<1 | 1
		a := absDiffU32(d0, uint32(target[c]))
		b := absDiffU32(d1, uint32(target[c]))
		err0 += a
		err1 += b
	}
	if err1 < err0 {
		return 1
	}
	return 0
}

func absDiffU32(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}
