package ddsutil

// This file implements BCn block compressors for encoding.
//
// The encoders are self-contained Go implementations; porting Intel's ISPC
// texture compressor was not practical. They produce valid, reasonable
// quality output tuned independently of the ISPC reference.
//
// Encoder strategy per format:
//   - bcFmt1: 4-color or 3-color + transparent endpoint search.
//   - bcFmt2: 4-bit rounded alpha + bcFmt1-style 4-color block.
//   - bcFmt3: 8-bit alpha endpoints with exact bcdec interpolation + bcFmt1 color.
//   - bcFmt4/bcFmt5: endpoint search with exact bcdec interpolation.
//   - BC6H: mode 11 (10.10.10 endpoints, no delta compression).
//   - bcFmt7: mode 6 (7-bit RGBA endpoints, p-bits, 4-bit indices).
//
// Quality selects how many endpoint candidates are evaluated per block:
//   - QualityFast: endpoints from the block's principal axis or value range.
//   - QualityNormal: QualityFast plus least squares refinement of the
//     endpoints, or a small search around the value range for alpha.
//   - QualitySlow: QualityNormal plus an exhaustive search over the block's
//     own values, all endpoint orientations for BC7 and coordinate descent
//     for BC6H.
//
// Blocks are independent, so block rows are compressed in parallel.

import "math"

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
	blocksX := int((width + blockWidth - 1) / blockWidth)
	blocksY := int((height + blockHeight - 1) / blockHeight)
	rowBytes := blocksX * blockSize
	out := make([]byte, blocksY*rowBytes)

	switch bc {
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
		parallelFor(blocksY, blocksX, func(by int) {
			y := uint32(by * blockHeight)
			for bx := 0; bx < blocksX; bx++ {
				x := uint32(bx * blockWidth)
				var block [16][4]uint16
				for j := uint32(0); j < 4; j++ {
					for i := uint32(0); i < 4; i++ {
						idx := ((y+j)*width + x + i) * 4
						block[j*4+i] = [4]uint16{halfData[idx], halfData[idx+1], halfData[idx+2], halfData[idx+3]}
					}
				}
				compressed := compressBc6Block(block, quality)
				copy(out[by*rowBytes+bx*blockSize:], compressed[:])
			}
		})
	default:
		rgba8, err := channelBytes(data)
		if err != nil {
			return nil, err
		}
		parallelFor(blocksY, blocksX, func(by int) {
			y := uint32(by * blockHeight)
			for bx := 0; bx < blocksX; bx++ {
				x := uint32(bx * blockWidth)
				var block [16][4]uint8
				for j := uint32(0); j < 4; j++ {
					for i := uint32(0); i < 4; i++ {
						idx := ((y+j)*width + x + i) * 4
						p := rgba8[idx : idx+4]
						block[j*4+i] = [4]uint8{p[0], p[1], p[2], p[3]}
					}
				}
				compressBlockU8(bc, &block, quality, out[by*rowBytes+bx*blockSize:])
			}
		})
	}

	return out, nil
}

// compressBlockU8 compresses an RGBA8 block into dst.
func compressBlockU8(bc bcFormat, block *[16][4]uint8, quality Quality, dst []byte) {
	switch bc {
	case bcFmt1:
		compressed := compressBc1Block(*block, quality)
		copy(dst, compressed[:])
	case bcFmt2:
		compressed := compressBc2Block(*block, quality)
		copy(dst, compressed[:])
	case bcFmt3:
		compressed := compressBc3Block(*block, quality)
		copy(dst, compressed[:])
	case bcFmt4, bcFmt4S:
		var values [16]uint8
		for i := 0; i < 16; i++ {
			values[i] = block[i][0]
		}
		compressed := compressBc4Block(values, bc == bcFmt4S, quality)
		copy(dst, compressed[:])
	case bcFmt5, bcFmt5S:
		var r, g [16]uint8
		for i := 0; i < 16; i++ {
			r[i] = block[i][0]
			g[i] = block[i][1]
		}
		compressed := compressBc5Block(r, g, bc == bcFmt5S, quality)
		copy(dst, compressed[:])
	case bcFmt7:
		compressed := compressBc7Block(*block, quality)
		copy(dst, compressed[:])
	}
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
// Shared helpers
// ---------------------------------------------------------------------------

// principalAxis returns the mean and the dominant direction of the first n
// channels of points. The axis is unit length, or zero if the points do not
// vary.
func principalAxis(points [][4]float32, n int) (mean, axis [4]float32) {
	if len(points) == 0 {
		return mean, axis
	}
	for _, p := range points {
		for c := 0; c < n; c++ {
			mean[c] += p[c]
		}
	}
	for c := 0; c < n; c++ {
		mean[c] /= float32(len(points))
	}

	// The covariance and power iteration use float64: products of HDR
	// values near the half float maximum overflow float32.
	var cov [4][4]float64
	for _, p := range points {
		for a := 0; a < n; a++ {
			da := float64(p[a] - mean[a])
			for b := 0; b < n; b++ {
				cov[a][b] += da * float64(p[b]-mean[b])
			}
		}
	}

	// Power iteration starting from the column with the largest variance.
	start := 0
	for c := 1; c < n; c++ {
		if cov[c][c] > cov[start][start] {
			start = c
		}
	}
	if cov[start][start] <= 0 {
		return mean, axis
	}
	v := cov[start]
	if !normalize(&v, n) {
		return mean, axis
	}
	for iteration := 0; iteration < 8; iteration++ {
		var next [4]float64
		for a := 0; a < n; a++ {
			for b := 0; b < n; b++ {
				next[a] += cov[a][b] * v[b]
			}
		}
		if !normalize(&next, n) {
			return mean, axis
		}
		v = next
	}
	for c := 0; c < n; c++ {
		axis[c] = float32(v[c])
	}
	return mean, axis
}

// normalize scales the first n components of v to unit length. It returns
// false if v has no finite, nonzero length.
func normalize(v *[4]float64, n int) bool {
	var norm float64
	for c := 0; c < n; c++ {
		norm += v[c] * v[c]
	}
	if !(norm > 0) || math.IsInf(norm, 0) {
		return false
	}
	norm = math.Sqrt(norm)
	for c := 0; c < n; c++ {
		v[c] /= norm
	}
	return true
}

// axisExtremes returns the points on the principal axis that bound the
// projections of points.
func axisExtremes(points [][4]float32, n int) (lo, hi [4]float32) {
	mean, axis := principalAxis(points, n)
	minT, maxT := float32(0), float32(0)
	for i, p := range points {
		var t float32
		for c := 0; c < n; c++ {
			t += (p[c] - mean[c]) * axis[c]
		}
		if i == 0 || t < minT {
			minT = t
		}
		if i == 0 || t > maxT {
			maxT = t
		}
	}
	for c := 0; c < n; c++ {
		lo[c] = mean[c] + minT*axis[c]
		hi[c] = mean[c] + maxT*axis[c]
	}
	return lo, hi
}

// leastSquaresEndpoints solves for the endpoints e0, e1 minimizing
// sum((1-w)*e0 + w*e1 - x)^2 per channel, given each point's weight w on e1.
// It returns false if the weights do not determine both endpoints.
func leastSquaresEndpoints(points [][4]float32, weights []float32, n int) (e0, e1 [4]float32, ok bool) {
	var aa, ab, bb float32
	var ax, bx [4]float32
	for i, p := range points {
		w := weights[i]
		a := 1 - w
		aa += a * a
		ab += a * w
		bb += w * w
		for c := 0; c < n; c++ {
			ax[c] += a * p[c]
			bx[c] += w * p[c]
		}
	}
	det := aa*bb - ab*ab
	if det > -1e-6 && det < 1e-6 {
		return e0, e1, false
	}
	for c := 0; c < n; c++ {
		e0[c] = (ax[c]*bb - bx[c]*ab) / det
		e1[c] = (bx[c]*aa - ax[c]*ab) / det
	}
	return e0, e1, true
}

// roundClamp rounds v to the nearest integer in [0, maxValue].
func roundClamp(v float32, maxValue int) int {
	q := int(math.Floor(float64(v) + 0.5))
	if q < 0 {
		return 0
	}
	if q > maxValue {
		return maxValue
	}
	return q
}

// quantize maps v in [0, 255] to the nearest of 0..maxValue.
func quantize(v float32, maxValue int) int {
	return roundClamp(v*float32(maxValue)/255.0, maxValue)
}

func absI32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func absDiffU32(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
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

// quantize565 rounds an RGB color in [0, 255] to the nearest 565 color.
func quantize565(c [4]float32) rgb565 {
	return rgb565{uint8(quantize(c[0], 31)), uint8(quantize(c[1], 63)), uint8(quantize(c[2], 31))}
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

// bc1Fit is a candidate bcFmt1 color block and its squared error.
type bc1Fit struct {
	c0, c1  rgb565
	indices [16]uint8
	err     uint64
}

// bc1Layout selects the palette layouts a color block may decode with.
type bc1Layout uint8

const (
	// bc1Opaque is a BC1 block without transparent pixels: the decoder uses
	// the 4-color layout when c0 > c1 and the 3-color layout otherwise.
	bc1Opaque bc1Layout = iota
	// bc1Transparent is a BC1 block with pixels with alpha < 128, which
	// requires the 3-color + transparent layout (c0 <= c1).
	bc1Transparent
	// bc1FourColor is a BC2/BC3 color block, which the decoder always
	// interprets with the 4-color layout.
	bc1FourColor
)

// fourColor reports whether a block with endpoints c0, c1 decodes with the
// 4-color layout.
func (layout bc1Layout) fourColor(c0, c1 rgb565) bool {
	return layout == bc1FourColor || (layout == bc1Opaque && pack565(c0) > pack565(c1))
}

// evaluateBc1 assigns the best index to every pixel for the endpoints c0, c1.
// It returns false if transparent pixels require the 3-color layout but
// c0 > c1.
func evaluateBc1(block *[16][4]uint8, c0, c1 rgb565, layout bc1Layout) (bc1Fit, bool) {
	hasTransparent := layout == bc1Transparent
	if hasTransparent && pack565(c0) > pack565(c1) {
		// Transparent mode requires c0 <= c1.
		return bc1Fit{}, false
	}
	useFourColor := layout.fourColor(c0, c1)

	var palette [4][3]uint8
	if useFourColor {
		palette = interpolateColor4(c0, c1)
	} else {
		palette = interpolateColor3(c0, c1)
	}
	maxK := 3
	if !useFourColor {
		// Index 3 is transparent black in the 3-color layout.
		maxK = 2
	}

	fit := bc1Fit{c0: c0, c1: c1}
	for i := 0; i < 16; i++ {
		if hasTransparent && block[i][3] < 128 {
			// Transparent pixels must use index 3.
			fit.indices[i] = 3
			continue
		}
		color := [3]uint8{block[i][0], block[i][1], block[i][2]}
		best := 0
		bestD := colorDistSq(palette[0], color)
		for k := 1; k <= maxK; k++ {
			if d := colorDistSq(palette[k], color); d < bestD {
				bestD = d
				best = k
			}
		}
		fit.indices[i] = uint8(best)
		fit.err += uint64(bestD)
	}
	return fit, true
}

// consider replaces best with the evaluation of (c0, c1) if it is better.
func (best *bc1Fit) consider(block *[16][4]uint8, c0, c1 rgb565, layout bc1Layout) {
	if fit, ok := evaluateBc1(block, c0, c1, layout); ok && fit.err < best.err {
		*best = fit
	}
}

// refineBc1 solves for the endpoints that best fit the current indices and
// keeps them while they reduce the error.
func refineBc1(block *[16][4]uint8, best *bc1Fit, layout bc1Layout) {
	points := make([][4]float32, 0, 16)
	weights := make([]float32, 0, 16)
	for iteration := 0; iteration < 4; iteration++ {
		fourColor := layout.fourColor(best.c0, best.c1)
		points, weights = points[:0], weights[:0]
		for i := 0; i < 16; i++ {
			var w float32
			switch idx := best.indices[i]; {
			case idx == 0:
				w = 0
			case idx == 1:
				w = 1
			case fourColor && idx == 2:
				w = 1.0 / 3.0
			case fourColor && idx == 3:
				w = 2.0 / 3.0
			case !fourColor && idx == 2:
				w = 0.5
			default:
				// Transparent or black pixels do not constrain the endpoints.
				continue
			}
			points = append(points, [4]float32{float32(block[i][0]), float32(block[i][1]), float32(block[i][2])})
			weights = append(weights, w)
		}
		e0, e1, ok := leastSquaresEndpoints(points, weights, 3)
		if !ok {
			return
		}
		c0, c1 := quantize565(e0), quantize565(e1)
		previous := best.err
		best.consider(block, c0, c1, layout)
		best.consider(block, c1, c0, layout)
		if best.err >= previous {
			return
		}
	}
}

// fitBc1Color finds the best (c0, c1) for the block with the given layout.
// Returns the packed color block.
func fitBc1Color(block [16][4]uint8, layout bc1Layout, quality Quality) uint64 {
	best := bc1Fit{err: ^uint64(0)}

	// Endpoints on the principal axis of the colors that are visible.
	points := make([][4]float32, 0, 16)
	for i := 0; i < 16; i++ {
		if layout != bc1Transparent || block[i][3] >= 128 {
			points = append(points, [4]float32{float32(block[i][0]), float32(block[i][1]), float32(block[i][2])})
		}
	}
	lo, hi := axisExtremes(points, 3)
	c0, c1 := quantize565(hi), quantize565(lo)
	best.consider(&block, c0, c1, layout)
	best.consider(&block, c1, c0, layout)

	if quality == QualitySlow {
		// Every pair of the unique quantized colors in the block.
		var unique [16]rgb565
		n := 0
		for i := 0; i < 16; i++ {
			c := rgb565{block[i][0] >> 3, block[i][1] >> 2, block[i][2] >> 3}
			duplicate := false
			for _, u := range unique[:n] {
				if u == c {
					duplicate = true
					break
				}
			}
			if !duplicate {
				unique[n] = c
				n++
			}
		}
		for _, c0 := range unique[:n] {
			for _, c1 := range unique[:n] {
				best.consider(&block, c0, c1, layout)
			}
		}
	}

	if quality != QualityFast {
		refineBc1(&block, &best, layout)
	}

	// Pack: colors first (4 bytes), then 2-bit indices (4 bytes).
	var packed uint64
	packed |= uint64(pack565(best.c0))
	packed |= uint64(pack565(best.c1)) << 16
	for i := 0; i < 16; i++ {
		packed |= uint64(best.indices[i]&3) << uint(32+i*2)
	}
	return packed
}

func compressBc1Block(block [16][4]uint8, quality Quality) [8]byte {
	hasTransparent := false
	for i := 0; i < 16; i++ {
		if block[i][3] < 128 {
			hasTransparent = true
			break
		}
	}
	layout := bc1Opaque
	if hasTransparent {
		layout = bc1Transparent
	}
	colorBlock := fitBc1Color(block, layout, quality)
	var out [8]byte
	for i := 0; i < 8; i++ {
		out[i] = byte(colorBlock >> uint(i*8))
	}
	return out
}

// ---------------------------------------------------------------------------
// bcFmt2
// ---------------------------------------------------------------------------

func compressBc2Block(block [16][4]uint8, quality Quality) [16]byte {
	// Alpha: 4 bits per pixel, rounded to the nearest of the 16 levels.
	var alpha uint64
	for i := 0; i < 16; i++ {
		alpha |= uint64((uint32(block[i][3])+8)/17) << uint(i*4)
	}
	colorBlock := fitBc1Color(block, bc1FourColor, quality)
	var out [16]byte
	for i := 0; i < 8; i++ {
		out[i] = byte(alpha >> uint(i*8))
		out[i+8] = byte(colorBlock >> uint(i*8))
	}
	return out
}

// ---------------------------------------------------------------------------
// Alpha endpoints (bcFmt3, bcFmt4, bcFmt5)
// ---------------------------------------------------------------------------

// alphaPalette computes the 8 palette entries for the endpoints a0, a1.
type alphaPalette func(a0, a1 int32) [8]int32

// alphaFit is a candidate alpha block and its squared error.
type alphaFit struct {
	e0, e1  int32
	indices uint64
	err     uint64
}

// consider replaces best with the evaluation of (a0, a1) if it is better.
func (best *alphaFit) consider(targets *[16]int32, palette alphaPalette, a0, a1 int32) {
	values := palette(a0, a1)
	var err uint64
	var indices uint64
	for i := 0; i < 16; i++ {
		bestK := 0
		bestD := absI32(values[0] - targets[i])
		for k := 1; k < 8; k++ {
			if d := absI32(values[k] - targets[i]); d < bestD {
				bestD = d
				bestK = k
			}
		}
		indices |= uint64(bestK) << uint(i*3)
		err += uint64(bestD) * uint64(bestD)
	}
	if err < best.err {
		*best = alphaFit{e0: a0, e1: a1, indices: indices, err: err}
	}
}

// fitAlphaEndpoints chooses two endpoints for 16 target values in [lo, hi],
// where lo and hi are also the fixed palette entries of the a0 <= a1 layout.
func fitAlphaEndpoints(targets *[16]int32, lo, hi int32, palette alphaPalette, quality Quality) alphaFit {
	best := alphaFit{err: ^uint64(0)}

	minV, maxV := targets[0], targets[0]
	for _, v := range targets[1:] {
		minV = min(minV, v)
		maxV = max(maxV, v)
	}

	// The value range with interpolated levels (a0 > a1), and the range of
	// the values that are not fixed palette entries (a0 <= a1).
	best.consider(targets, palette, maxV, minV)
	innerMin, innerMax := hi, lo
	for _, v := range targets {
		if v != lo && v != hi {
			innerMin = min(innerMin, v)
			innerMax = max(innerMax, v)
		}
	}
	if innerMin <= innerMax {
		best.consider(targets, palette, innerMin, innerMax)
	}
	if quality == QualityFast {
		return best
	}

	// Endpoints slightly inside or outside the value range can place the
	// interpolated levels closer to the values.
	window := int32(2)
	if quality == QualitySlow {
		window = 4
	}
	for d0 := -window; d0 <= window; d0++ {
		for d1 := -window; d1 <= window; d1++ {
			a0 := min(max(maxV+d0, lo), hi)
			a1 := min(max(minV+d1, lo), hi)
			best.consider(targets, palette, a0, a1)
			best.consider(targets, palette, a1, a0)
		}
	}

	if quality == QualitySlow {
		// Every pair of the distinct values.
		var unique [16]int32
		n := 0
		for _, v := range targets {
			duplicate := false
			for _, u := range unique[:n] {
				if u == v {
					duplicate = true
					break
				}
			}
			if !duplicate {
				unique[n] = v
				n++
			}
		}
		for _, a0 := range unique[:n] {
			for _, a1 := range unique[:n] {
				best.consider(targets, palette, a0, a1)
			}
		}
	}
	return best
}

// packAlphaBlock stores the endpoints and 3-bit indices of an alpha block.
func packAlphaBlock(fit alphaFit) [8]byte {
	var out [8]byte
	out[0] = byte(fit.e0)
	out[1] = byte(fit.e1)
	for i := 0; i < 6; i++ {
		out[2+i] = byte(fit.indices >> uint(i*8))
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

// smoothAlphaPalette uses the 6-level layout for a0 > a1 and the 4-level
// layout with fixed 0/255 entries for a0 <= a1.
func smoothAlphaPalette(a0, a1 int32) [8]int32 {
	var levels [8]uint32
	if a0 > a1 {
		levels = interpolateAlpha6(uint32(a0), uint32(a1))
	} else {
		levels = interpolateAlpha4(uint32(a0), uint32(a1))
	}
	var palette [8]int32
	for i, v := range levels {
		palette[i] = int32(v)
	}
	return palette
}

func compressSmoothAlpha(alphas [16]uint8, quality Quality) [8]byte {
	var targets [16]int32
	for i, a := range alphas {
		targets[i] = int32(a)
	}
	return packAlphaBlock(fitAlphaEndpoints(&targets, 0, 255, smoothAlphaPalette, quality))
}

func compressBc3Block(block [16][4]uint8, quality Quality) [16]byte {
	var alphas [16]uint8
	for i := 0; i < 16; i++ {
		alphas[i] = block[i][3]
	}
	alphaBlock := compressSmoothAlpha(alphas, quality)
	colorBlock := fitBc1Color(block, bc1FourColor, quality)
	var out [16]byte
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
func interpolateBc4(a0, a1 int32, isSigned bool) [8]int32 {
	var aWeights4 = [4]int32{13107, 26215, 39321, 52429}
	var aWeights6 = [6]int32{9363, 18724, 28086, 37450, 46812, 56173}

	var alpha [8]int32
	alpha[0] = a0
	alpha[1] = a1
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

func bc4UnsignedPalette(a0, a1 int32) [8]int32 { return interpolateBc4(a0, a1, false) }

func bc4SignedPalette(a0, a1 int32) [8]int32 { return interpolateBc4(a0, a1, true) }

// compressBc4Block encodes one channel of 16 values. For signed blocks the
// values are the bit patterns of snorm8 values (int8), as stored in the block.
func compressBc4Block(values [16]uint8, isSigned bool, quality Quality) [8]byte {
	var targets [16]int32
	for i := 0; i < 16; i++ {
		v := int32(values[i])
		if isSigned {
			v = int32(int8(values[i]))
			if v < -127 {
				// The decoder clamps -128 to -127.
				v = -127
			}
		}
		targets[i] = v
	}
	if isSigned {
		return packAlphaBlock(fitAlphaEndpoints(&targets, -127, 127, bc4SignedPalette, quality))
	}
	return packAlphaBlock(fitAlphaEndpoints(&targets, 0, 255, bc4UnsignedPalette, quality))
}

func compressBc5Block(r, g [16]uint8, isSigned bool, quality Quality) [16]byte {
	rBlock := compressBc4Block(r, isSigned, quality)
	gBlock := compressBc4Block(g, isSigned, quality)
	var out [16]byte
	copy(out[:8], rBlock[:])
	copy(out[8:], gBlock[:])
	return out
}

// ---------------------------------------------------------------------------
// BC6H
// ---------------------------------------------------------------------------

// bc6Weight4 is the 4-bit index weight table.
var bc6Weight4 = [16]int32{0, 4, 9, 13, 17, 21, 26, 30, 34, 38, 43, 47, 51, 55, 60, 64}

// bc6Fit is a candidate BC6H mode 11 block and its squared error.
type bc6Fit struct {
	e0, e1  [3]uint16
	indices uint64
	err     float64
}

// consider replaces best with the evaluation of (e0, e1) if it is better.
//
// The decoder interpolates the unquantized endpoint values (e*64+32) and
// scales by 31/64, giving half bits h = interp*31 + 15. Compare in float
// space.
func (best *bc6Fit) consider(targets *[16][3]float32, e0, e1 [3]uint16) {
	var palette [16][3]float32
	for k := 0; k < 16; k++ {
		for c := 0; c < 3; c++ {
			interp := interpolateI32(int32(e0[c]), int32(e1[c]), bc6Weight4[k])
			h := (interp*1984 + 992) >> 6 // finish_unquantize unsigned
			palette[k][c] = halfToFloat32(uint16(h))
		}
	}

	var err float64
	var indices uint64
	for i := 0; i < 16; i++ {
		bestK := 0
		bestErr := float64(0xFFFFFFFFFFFFFFFF)
		for k := 0; k < 16; k++ {
			var e float64
			for c := 0; c < 3; c++ {
				d := float64(palette[k][c] - targets[i][c])
				e += d * d
			}
			if e < bestErr {
				bestErr = e
				bestK = k
			}
		}
		indices |= uint64(bestK) << uint(i*4)
		err += bestErr
	}
	if err < best.err {
		*best = bc6Fit{e0: e0, e1: e1, indices: indices, err: err}
	}
}

// compressBc6Block encodes a block using BC6H mode 11 (0b00011):
// no partition, both endpoints stored explicitly with 10 bits each.
func compressBc6Block(block [16][4]uint16, quality Quality) [16]byte {
	// For unsigned mode, clamp negative halves (sign bit set) to zero.
	var minV [3]uint16
	var maxV [3]uint16
	for c := 0; c < 3; c++ {
		minV[c] = 0xFFFF
		maxV[c] = 0
	}
	var targets [16][3]float32
	for i := 0; i < 16; i++ {
		for c := 0; c < 3; c++ {
			v := block[i][c] & 0x7FFF
			targets[i][c] = halfToFloat32(v)
			if v < minV[c] {
				minV[c] = v
			}
			if v > maxV[c] {
				maxV[c] = v
			}
		}
	}

	// Decoded half bits for a 10-bit endpoint e are approximately e*31.
	// Invert: e = round(h / 31). Each channel maps its min to either
	// endpoint; the orientation selects which.
	endpoints := func(orientation int) (e0, e1 [3]uint16) {
		for c := 0; c < 3; c++ {
			lo := clampU16(uint16((uint32(minV[c])+15)/31), 1023)
			hi := clampU16(uint16((uint32(maxV[c])+15)/31), 1023)
			if (orientation>>c)&1 == 1 {
				e0[c], e1[c] = hi, lo
			} else {
				e0[c], e1[c] = lo, hi
			}
		}
		return e0, e1
	}

	best := bc6Fit{err: math.Inf(1)}
	if quality == QualityFast {
		// Flip the channels that decrease along the principal axis.
		points := make([][4]float32, 16)
		for i := range points {
			points[i] = [4]float32{targets[i][0], targets[i][1], targets[i][2]}
		}
		_, axis := principalAxis(points, 3)
		orientation := 0
		for c := 0; c < 3; c++ {
			if axis[c] < 0 {
				orientation |= 1 << c
			}
		}
		e0, e1 := endpoints(orientation)
		best.consider(&targets, e0, e1)
	} else {
		for orientation := 0; orientation < 8; orientation++ {
			e0, e1 := endpoints(orientation)
			best.consider(&targets, e0, e1)
		}
	}

	if quality == QualitySlow {
		// Coordinate descent on the 6 endpoint components.
		for _, step := range []int32{16, 4, 1} {
			for improved := true; improved; {
				improved = false
				for endpoint := 0; endpoint < 2; endpoint++ {
					for c := 0; c < 3; c++ {
						for _, delta := range []int32{-step, step} {
							e0, e1 := best.e0, best.e1
							e := &e0
							if endpoint == 1 {
								e = &e1
							}
							v := int32(e[c]) + delta
							if v < 0 || v > 1023 {
								continue
							}
							e[c] = uint16(v)
							previous := best.err
							best.consider(&targets, e0, e1)
							if best.err < previous {
								improved = true
							}
						}
					}
				}
			}
		}
	}

	// The fix-up index is stored without its most significant bit, so it
	// must be below 8. Swapping the endpoints and inverting the indices
	// produces the same palette with the fix-up index in range.
	if best.indices&0xF >= 8 {
		best.e0, best.e1 = best.e1, best.e0
		best.indices = ^best.indices
	}

	// Assemble the bitstream LSB-first.
	// Mode 11: 0b00011 LSB-first: 1, 1, 0, 0, 0; endpoints; then indices
	// (first pixel uses 3 bits as the fix-up, the rest 4 bits).
	return assembleBc6Mode11(best.e0, best.e1, best.indices)
}

// bitWriter writes bits LSB-first into a 128-bit block.
type bitWriter struct {
	words  [16]byte
	bitPos uint
}

func (w *bitWriter) write(value uint64, count uint) {
	for i := uint(0); i < count; i++ {
		if (value>>i)&1 == 1 {
			w.words[w.bitPos/8] |= 1 << (w.bitPos % 8)
		}
		w.bitPos++
	}
}

// writeIndices writes 4-bit indices where the fix-up pixel (0) uses 3 bits.
// The fix-up's 4th bit is dropped, so write the stream per pixel.
func (w *bitWriter) writeIndices(indices uint64) {
	for i := 0; i < 16; i++ {
		count := uint(4)
		if i == 0 {
			count = 3
		}
		w.write((indices>>uint(i*4))&((1<<count)-1), count)
	}
}

// assembleBc6Mode11 builds the 128-bit mode 11 block.
func assembleBc6Mode11(e0, e1 [3]uint16, indices uint64) [16]byte {
	var w bitWriter
	// Mode 11: 0b00011 LSB-first: bits 0,1 = 1,1; bits 2-4 = 0,0,0
	w.write(0b11, 2)
	w.write(0, 3)
	for c := 0; c < 3; c++ {
		w.write(uint64(e0[c]), 10)
	}
	for c := 0; c < 3; c++ {
		w.write(uint64(e1[c]), 10)
	}
	w.writeIndices(indices)
	return w.words
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

// bc7Fit is a candidate BC7 mode 6 block and its squared error.
type bc7Fit struct {
	e0, e1  [4]uint8
	p0, p1  uint8
	indices uint64
	err     uint64
}

// consider replaces best with the evaluation of the 7-bit endpoints e0, e1
// and p-bits p0, p1 if it is better.
func (best *bc7Fit) consider(block *[16][4]uint8, e0, e1 [4]uint8, p0, p1 uint8) {
	// Decoded endpoint values (8-bit) and the interpolated palette.
	var palette [16][4]uint32
	for c := 0; c < 4; c++ {
		d0 := uint32(e0[c])<<1 | uint32(p0)
		d1 := uint32(e1[c])<<1 | uint32(p1)
		for k := 0; k < 16; k++ {
			w := bc7Weight4[k]
			palette[k][c] = (d0*(64-w) + d1*w + 32) >> 6
		}
	}

	// Assign 4-bit indices and accumulate the error.
	var err uint64
	var indices uint64
	for i := 0; i < 16; i++ {
		bestK := 0
		bestErr := uint64(0xFFFFFFFFFFFFFFFF)
		for k := 0; k < 16; k++ {
			var e uint64
			for c := 0; c < 4; c++ {
				d := uint64(absDiffU32(palette[k][c], uint32(block[i][c])))
				e += d * d
			}
			if e < bestErr {
				bestErr = e
				bestK = k
			}
		}
		indices |= uint64(bestK) << uint(i*4)
		err += bestErr
	}
	if err < best.err {
		*best = bc7Fit{e0: e0, e1: e1, p0: p0, p1: p1, indices: indices, err: err}
	}
}

// quantizeBc7Endpoint returns the 7-bit components and p-bit that decode
// closest to the 8-bit endpoint d for the given p-bit. A component decodes
// as (e<<1)|p.
func quantizeBc7Endpoint(d [4]float32, p uint8) (e [4]uint8, err float32) {
	for c := 0; c < 4; c++ {
		e[c] = uint8(roundClamp((d[c]-float32(p))/2, 127))
		diff := float32(uint32(e[c])<<1|uint32(p)) - d[c]
		err += diff * diff
	}
	return e, err
}

// considerDecoded quantizes the decoded 8-bit endpoints d0, d1 and considers
// the result. With allPBits, every p-bit combination is considered;
// otherwise each endpoint uses the p-bit that quantizes it best.
func (best *bc7Fit) considerDecoded(block *[16][4]uint8, d0, d1 [4]float32, allPBits bool) {
	if allPBits {
		for p0 := uint8(0); p0 < 2; p0++ {
			for p1 := uint8(0); p1 < 2; p1++ {
				e0, _ := quantizeBc7Endpoint(d0, p0)
				e1, _ := quantizeBc7Endpoint(d1, p1)
				best.consider(block, e0, e1, p0, p1)
			}
		}
		return
	}
	pick := func(d [4]float32) ([4]uint8, uint8) {
		e0, err0 := quantizeBc7Endpoint(d, 0)
		e1, err1 := quantizeBc7Endpoint(d, 1)
		if err1 < err0 {
			return e1, 1
		}
		return e0, 0
	}
	e0, p0 := pick(d0)
	e1, p1 := pick(d1)
	best.consider(block, e0, e1, p0, p1)
}

// refine solves for the endpoints that best fit the current indices and
// keeps them while they reduce the error.
func (best *bc7Fit) refine(block *[16][4]uint8, points [][4]float32, allPBits bool) {
	weights := make([]float32, 16)
	for iteration := 0; iteration < 4; iteration++ {
		for i := range weights {
			weights[i] = float32(bc7Weight4[(best.indices>>uint(i*4))&0xF]) / 64.0
		}
		d0, d1, ok := leastSquaresEndpoints(points, weights, 4)
		if !ok {
			return
		}
		previous := best.err
		best.considerDecoded(block, d0, d1, allPBits)
		if best.err >= previous {
			return
		}
	}
}

// compressBc7Block encodes a block using BC7 mode 6:
// 1 subset, 7-bit RGBA endpoints with unique p-bits, 4-bit indices.
func compressBc7Block(block [16][4]uint8, quality Quality) [16]byte {
	// Per-channel min/max of the block.
	var minV, maxV [4]uint8
	for c := 0; c < 4; c++ {
		minV[c] = 255
		maxV[c] = 0
	}
	points := make([][4]float32, 16)
	for i := 0; i < 16; i++ {
		for c := 0; c < 4; c++ {
			points[i][c] = float32(block[i][c])
			if block[i][c] < minV[c] {
				minV[c] = block[i][c]
			}
			if block[i][c] > maxV[c] {
				maxV[c] = block[i][c]
			}
		}
	}

	// Endpoints are per-channel independent, so each channel can map its
	// min to either endpoint. The orientation selects which.
	best := bc7Fit{err: ^uint64(0)}
	tryOrientation := func(fit *bc7Fit, orientation int) {
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
		fit.consider(&block, e0, e1, bestPBit(e0, t0), bestPBit(e1, t1))
	}

	// Flip the channels that decrease along the principal axis.
	_, axis := principalAxis(points, 4)
	orientation := 0
	for c := 0; c < 4; c++ {
		if axis[c] < 0 {
			orientation |= 1 << c
		}
	}
	tryOrientation(&best, orientation)
	if quality != QualityFast {
		best.refine(&block, points, false)
	}

	if quality == QualitySlow {
		// Refine the best of all orientations as well, keeping the better
		// of the two results.
		exhaustive := bc7Fit{err: ^uint64(0)}
		for orientation := 0; orientation < 16; orientation++ {
			tryOrientation(&exhaustive, orientation)
		}
		exhaustive.refine(&block, points, true)
		best.refine(&block, points, true)
		if exhaustive.err < best.err {
			best = exhaustive
		}
	}

	// The fix-up index is stored without its most significant bit, so it
	// must be below 8. The weights are symmetric (w[15-k] = 64 - w[k]), so
	// swapping the endpoints and inverting the indices decodes identically.
	if best.indices&0xF >= 8 {
		best.e0, best.e1 = best.e1, best.e0
		best.p0, best.p1 = best.p1, best.p0
		best.indices = ^best.indices
	}

	// Assemble the bitstream LSB-first.
	var w bitWriter
	// Mode 6: six zeros then a 1.
	w.write(0, 6)
	w.write(1, 1)
	// Endpoints are stored channel-major (matching the decoder):
	// R0, R1, G0, G1, B0, B1, A0, A1.
	for c := 0; c < 4; c++ {
		w.write(uint64(best.e0[c]), 7)
		w.write(uint64(best.e1[c]), 7)
	}
	// P-bits.
	w.write(uint64(best.p0), 1)
	w.write(uint64(best.p1), 1)
	w.writeIndices(best.indices)
	return w.words
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
