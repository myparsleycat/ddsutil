package ddsutil

// maxMipmapCount returns log2(max_dimension) + 1, or 0 for a zero dimension.
func maxMipmapCount(maxDimension uint32) uint32 {
	if maxDimension == 0 {
		return 0
	}
	// log2(x) + 1
	return 32 - uint32(leadingZeros32(maxDimension))
}

func leadingZeros32(x uint32) int {
	n := 0
	if x>>16 == 0 {
		n += 16
		x <<= 16
	}
	if x>>24 == 0 {
		n += 8
		x <<= 8
	}
	if x>>28 == 0 {
		n += 4
		x <<= 4
	}
	if x>>30 == 0 {
		n += 2
		x <<= 2
	}
	if x>>31 == 0 {
		n++
	}
	return n
}

// MipDimension returns the reduced value for baseDimension at level mipmap.
func MipDimension(baseDimension uint32, mipmap uint32) uint32 {
	// Halve for each mip level.
	v := baseDimension >> mipmap
	if v < 1 {
		return 1
	}
	return v
}

// mipSize computes the size in bytes of a mip level with the given dimensions
// and block properties, or ok=false on overflow.
func mipSize(width, height, depth, blockWidth, blockHeight, blockDepth, blockSizeInBytes int) (int, bool) {
	w := (width + blockWidth - 1) / blockWidth
	h := (height + blockHeight - 1) / blockHeight
	d := (depth + blockDepth - 1) / blockDepth
	v, ok := mulOverflow(w, h)
	if !ok {
		return 0, false
	}
	v, ok = mulOverflow(v, d)
	if !ok {
		return 0, false
	}
	v, ok = mulOverflow(v, blockSizeInBytes)
	if !ok {
		return 0, false
	}
	return v, true
}

func mulOverflow(a, b int) (int, bool) {
	if a == 0 || b == 0 {
		return 0, true
	}
	c := a * b
	if c/b != a {
		return 0, false
	}
	return c, true
}

// calculateOffset returns the byte offset of (layer, depth_level, mipmap)
// within a tightly packed surface, or ok=false when the request is out of
// bounds or would overflow.
func calculateOffset(layer, depthLevel, mipmap uint32, dimensions [3]uint32, blockDimensions [3]uint32, blockSizeInBytes int, mipmapsPerLayer uint32) (int, bool) {
	width, height, depth := dimensions[0], dimensions[1], dimensions[2]
	blockWidth, blockHeight, blockDepth := blockDimensions[0], blockDimensions[1], blockDimensions[2]

	mipSizes := make([]int, 0, mipmapsPerLayer)
	for i := uint32(0); i < mipmapsPerLayer; i++ {
		mipW := MipDimension(width, i)
		mipH := MipDimension(height, i)
		mipD := MipDimension(depth, i)
		size, ok := mipSize(int(mipW), int(mipH), int(mipD), int(blockWidth), int(blockHeight), int(blockDepth), blockSizeInBytes)
		if !ok {
			return 0, false
		}
		mipSizes = append(mipSizes, size)
	}

	// Each depth level adds another rounded 2D slice.
	mipWidth := MipDimension(width, mipmap)
	mipHeight := MipDimension(height, mipmap)
	mipSize2d, ok := mipSize(int(mipWidth), int(mipHeight), 1, int(blockWidth), int(blockHeight), int(blockDepth), blockSizeInBytes)
	if !ok {
		return 0, false
	}

	// Assume mipmaps are tightly packed. This is the case for DDS surface data.
	layerSize := 0
	for _, s := range mipSizes {
		layerSize += s
	}

	// Each layer should have the same number of mipmaps.
	layerOffset, ok := mulOverflow(int(layer), layerSize)
	if !ok {
		return 0, false
	}
	if int(mipmap) > len(mipSizes) {
		return 0, false
	}
	mipOffset := 0
	for _, s := range mipSizes[:mipmap] {
		mipOffset += s
	}
	depthOffset, ok := mulOverflow(mipSize2d, int(depthLevel))
	if !ok {
		return 0, false
	}
	total, ok := addOverflow3(layerOffset, mipOffset, depthOffset)
	if !ok {
		return 0, false
	}
	return total, true
}

func addOverflow3(a, b, c int) (int, bool) {
	s := a + b
	if (s < a) != (b < 0) {
		return 0, false
	}
	t := s + c
	if (t < s) != (c < 0) {
		return 0, false
	}
	return t, true
}

// downsampleRgba halves the width and height by averaging pixels
// (a 2x2x2 region averages into a 1x1x1 region).
//
// The average accumulates in float32 and rounds at each step (f32 sum, f32
// division) for deterministic, reproducible results.
func downsampleRgba[T channel](newWidth, newHeight, newDepth, width, height, depth int, data []T) []T {
	newData := make([]T, newWidth*newHeight*newDepth*4)
	for z := 0; z < newDepth; z++ {
		for x := 0; x < newWidth; x++ {
			for y := 0; y < newHeight; y++ {
				newIndex := (z*newWidth*newHeight + y*newWidth + x)
				for c := 0; c < 4; c++ {
					var sum float32
					var count uint64
					for z2 := 0; z2 < 2; z2++ {
						sampledZ := (z * 2) + z2
						if sampledZ < depth {
							for y2 := 0; y2 < 2; y2++ {
								sampledY := (y * 2) + y2
								if sampledY < height {
									for x2 := 0; x2 < 2; x2++ {
										sampledX := (x * 2) + x2
										if sampledX < width {
											index := (sampledZ*width*height + sampledY*width + sampledX)
											sum += toF32(data[index*4+c])
											count++
										}
									}
								}
							}
						}
					}
					div := count
					if div < 1 {
						div = 1
					}
					newData[newIndex*4+c] = fromF32[T](sum / float32(div))
				}
			}
		}
	}
	return newData
}
