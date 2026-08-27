package ddsutil

// All BCN formats use 4x4 pixel blocks.
const (
	blockWidth       = 4
	blockHeight      = 4
	channels         = 4
	elementsPerBlock = blockWidth * blockHeight * channels
)

// decodeBcn decompresses the bytes in data to the uncompressed RGBA8 format.
func decodeBcn(bc bcFormat, width, height uint32, data []byte) ([]uint8, error) {
	// Validate surface dimensions to check for potential overflow.
	expectedSize, ok := mipSize(int(width), int(height), 1, blockWidth, blockHeight, 1, bc.blockSizeInBytes())
	if !ok {
		return nil, &SurfaceError{Kind: SurfaceErrorPixelCountWouldOverflow, Width: width, Height: height, Depth: 1}
	}

	// Mipmap dimensions do not need to be multiples of the block dimensions.
	// A mipmap of size 1x1 pixels can still be decoded.
	// Simply checking the data length is sufficient.
	if len(data) < expectedSize {
		return nil, &SurfaceError{Kind: SurfaceErrorNotEnoughData, Expected: expectedSize, Actual: len(data)}
	}

	rgba := make([]uint8, int(width)*int(height)*channels)

	// BCN formats lay out blocks in row-major order.
	blockStart := 0
	blockSize := bc.blockSizeInBytes()
	for y := uint32(0); y < height; y += blockHeight {
		for x := uint32(0); x < width; x += blockWidth {
			block := data[blockStart : blockStart+blockSize]
			pixels := decompressBlock(bc, block)
			putRgbaBlockU8(rgba, &pixels, int(x), int(y), int(width), int(height))
			blockStart += blockSize
		}
	}

	return rgba, nil
}

// decodeBcnF32 decompresses the bytes in data to the uncompressed RGBAF32 format.
func decodeBcnF32(bc bcFormat, width, height uint32, data []byte) ([]float32, error) {
	expectedSize, ok := mipSize(int(width), int(height), 1, blockWidth, blockHeight, 1, bc.blockSizeInBytes())
	if !ok {
		return nil, &SurfaceError{Kind: SurfaceErrorPixelCountWouldOverflow, Width: width, Height: height, Depth: 1}
	}
	if len(data) < expectedSize {
		return nil, &SurfaceError{Kind: SurfaceErrorNotEnoughData, Expected: expectedSize, Actual: len(data)}
	}

	rgba := make([]float32, int(width)*int(height)*channels)

	blockStart := 0
	blockSize := bc.blockSizeInBytes()
	for y := uint32(0); y < height; y += blockHeight {
		for x := uint32(0); x < width; x += blockWidth {
			block := data[blockStart : blockStart+blockSize]
			pixels := decompressBlockF32(bc, block)
			putRgbaBlockF32(rgba, &pixels, int(x), int(y), int(width), int(height))
			blockStart += blockSize
		}
	}

	return rgba, nil
}

// bcFormat identifies a BCn compressed format for block decode/encode.
type bcFormat uint8

const (
	bcFmt1 bcFormat = iota
	bcFmt2
	bcFmt3
	bcFmt4
	bcFmt4S
	bcFmt5
	bcFmt5S
	bcFmt6
	bcFmt7
)

func (f bcFormat) blockSizeInBytes() int {
	switch f {
	case bcFmt1, bcFmt4, bcFmt4S:
		return 8
	default:
		return 16
	}
}

// decompressBlock decodes a single 4x4 block to rgba8.
func decompressBlock(f bcFormat, block []byte) [blockHeight][blockWidth][4]uint8 {
	var out [blockHeight][blockWidth][4]uint8
	var flat [blockWidth * blockHeight * 4]uint8
	switch f {
	case bcFmt1:
		bc1(block, flat[:], blockWidth*channels)
	case bcFmt2:
		bc2(block, flat[:], blockWidth*channels)
	case bcFmt3:
		bc3(block, flat[:], blockWidth*channels)
	case bcFmt4:
		// bcFmt4 stores grayscale data, so each decompressed pixel is 1 byte.
		bc4(block, flat[:16], blockWidth, false)
		for i := 0; i < 16; i++ {
			v := flat[i]
			out[i/4][i%4] = [4]uint8{v, v, v, 255}
		}
		return out
	case bcFmt4S:
		bc4(block, flat[:16], blockWidth, true)
		for i := 0; i < 16; i++ {
			v := snorm8ToUnorm8(flat[i])
			out[i/4][i%4] = [4]uint8{v, v, v, 255}
		}
		return out
	case bcFmt5:
		// bcFmt5 stores RG data, so each decompressed pixel is 2 bytes.
		bc5(block, flat[:32], blockWidth*2, false)
		for i := 0; i < 16; i++ {
			// It's convention to zero the blue channel when decompressing bcFmt5.
			out[i/4][i%4] = [4]uint8{flat[i*2], flat[i*2+1], 0, 255}
		}
		return out
	case bcFmt5S:
		bc5(block, flat[:32], blockWidth*2, true)
		for i := 0; i < 16; i++ {
			out[i/4][i%4] = [4]uint8{
				snorm8ToUnorm8(flat[i*2]),
				snorm8ToUnorm8(flat[i*2+1]),
				snorm8ToUnorm8(0),
				255,
			}
		}
		return out
	case bcFmt6:
		var rgb [blockWidth * blockHeight * 3]float32
		bc6hFloat(block, rgb[:], blockWidth*3, false)
		for i := 0; i < 16; i++ {
			// Truncate to clamp to 0 to 255.
			out[i/4][i%4] = [4]uint8{
				f32ToU8(rgb[i*3] * 255.0),
				f32ToU8(rgb[i*3+1] * 255.0),
				f32ToU8(rgb[i*3+2] * 255.0),
				255,
			}
		}
		return out
	case bcFmt7:
		bc7(block, flat[:], blockWidth*channels)
	}
	// Copy the flat block into the nested array.
	for i := 0; i < 16; i++ {
		out[i/4][i%4] = [4]uint8{flat[i*4], flat[i*4+1], flat[i*4+2], flat[i*4+3]}
	}
	return out
}

// decompressBlockF32 decodes a single 4x4 block to rgba f32.
func decompressBlockF32(f bcFormat, block []byte) [blockHeight][blockWidth][4]float32 {
	var out [blockHeight][blockWidth][4]float32
	switch f {
	case bcFmt4S:
		var flat [blockWidth * blockHeight]float32
		bc4Float(block, flat[:], blockWidth, true)
		for i := 0; i < 16; i++ {
			v := flat[i]
			out[i/4][i%4] = [4]float32{v, v, v, 1.0}
		}
	case bcFmt5S:
		var flat [blockWidth * blockHeight * 2]float32
		bc5Float(block, flat[:], blockWidth*2, true)
		for i := 0; i < 16; i++ {
			out[i/4][i%4] = [4]float32{flat[i*2], flat[i*2+1], 0.5, 1.0}
		}
	case bcFmt6:
		var flat [blockWidth * blockHeight * 3]float32
		bc6hFloat(block, flat[:], blockWidth*3, false)
		for i := 0; i < 16; i++ {
			out[i/4][i%4] = [4]float32{flat[i*3], flat[i*3+1], flat[i*3+2], 1.0}
		}
	}
	return out
}

// putRgbaBlockU8 writes a decoded 4x4 pixel block into the surface at (x, y),
// clipping partial blocks at the edges.
func putRgbaBlockU8(surface []uint8, pixels *[blockHeight][blockWidth][4]uint8, x, y, width, height int) {
	// Place the compressed block into the decompressed surface.
	// The data from each block will update up to 4 rows of the RGBA surface.
	// Add checks since the edges won't always have full blocks.
	if x > width || y > height {
		return
	}
	elementsPerRow := channels * minInt(blockWidth, width-x)
	rows := minInt(blockHeight, height-y)
	for row := 0; row < rows; row++ {
		surfaceIndex := ((y+row)*width + x) * channels
		rowPixels := pixels[row]
		for px := 0; px < elementsPerRow/4; px++ {
			p := rowPixels[px]
			surface[surfaceIndex+px*4] = p[0]
			surface[surfaceIndex+px*4+1] = p[1]
			surface[surfaceIndex+px*4+2] = p[2]
			surface[surfaceIndex+px*4+3] = p[3]
		}
	}
}

// putRgbaBlockF32 writes a decoded 4x4 float pixel block into the surface.
func putRgbaBlockF32(surface []float32, pixels *[blockHeight][blockWidth][4]float32, x, y, width, height int) {
	if x > width || y > height {
		return
	}
	elementsPerRow := channels * minInt(blockWidth, width-x)
	rows := minInt(blockHeight, height-y)
	for row := 0; row < rows; row++ {
		surfaceIndex := ((y+row)*width + x) * channels
		rowPixels := pixels[row]
		for px := 0; px < elementsPerRow/4; px++ {
			p := rowPixels[px]
			surface[surfaceIndex+px*4] = p[0]
			surface[surfaceIndex+px*4+1] = p[1]
			surface[surfaceIndex+px*4+2] = p[2]
			surface[surfaceIndex+px*4+3] = p[3]
		}
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
