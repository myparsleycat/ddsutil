package ddsutil

// encodeMipmapsOptions describes the per-layer encode loop state.
type mipData[T channel] struct {
	width  int
	height int
	depth  int
	data   []T
}

// downsample halves the mip data dimensions using the base surface dimensions.
func (m *mipData[T]) downsample(baseWidth, baseHeight, baseDepth uint32, blockDimensions [3]uint32, mipmap uint32) *mipData[T] {
	// Mip dimensions are the padded virtual size of the mipmap.
	// Padding the physical size of the previous mip produces incorrect results.
	width, height, depth := physicalDimensions(
		MipDimension(baseWidth, mipmap),
		MipDimension(baseHeight, mipmap),
		MipDimension(baseDepth, mipmap),
		blockDimensions,
	)

	// Assume the data is already padded.
	data := downsampleRgba(width, height, depth, m.width, m.height, m.depth, m.data)

	return &mipData[T]{width: width, height: height, depth: depth, data: data}
}

// encode encodes the mip data to the target format.
func (m *mipData[T]) encode(format ImageFormat, quality Quality) ([]byte, error) {
	return encodePixels[T](uint32(m.width), uint32(m.height)*uint32(m.depth), m.data, format, quality)
}

// Encode encodes an RGBA8 surface to the given format.
// The number of mipmaps generated depends on the mipmaps parameter.
func (s *SurfaceRgba8) Encode(format ImageFormat, quality Quality, mipmaps Mipmaps) (*Surface, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return encodeSurface(s, format, quality, mipmaps)
}

// Encode encodes an RGBAF32 surface to the given format.
// The number of mipmaps generated depends on the mipmaps parameter.
func (s *SurfaceRgba32Float) Encode(format ImageFormat, quality Quality, mipmaps Mipmaps) (*Surface, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return encodeSurface(s, format, quality, mipmaps)
}

// getMipmapSurface is the interface used by encodeSurface.
type getMipmapSurface[P any] interface {
	surfaceWidth() uint32
	surfaceHeight() uint32
	surfaceDepth() uint32
	surfaceLayers() uint32
	surfaceMipmaps() uint32
	get(layer, depthLevel, mipmap uint32) []P
}

func (s *SurfaceRgba8) surfaceWidth() uint32       { return s.Width }
func (s *SurfaceRgba8) surfaceHeight() uint32      { return s.Height }
func (s *SurfaceRgba8) surfaceDepth() uint32       { return s.Depth }
func (s *SurfaceRgba8) surfaceLayers() uint32      { return s.Layers }
func (s *SurfaceRgba8) surfaceMipmaps() uint32     { return s.Mipmaps }
func (s *SurfaceRgba8) get(l, d, m uint32) []uint8 { return s.Get(l, d, m) }

func (s *SurfaceRgba32Float) surfaceWidth() uint32         { return s.Width }
func (s *SurfaceRgba32Float) surfaceHeight() uint32        { return s.Height }
func (s *SurfaceRgba32Float) surfaceDepth() uint32         { return s.Depth }
func (s *SurfaceRgba32Float) surfaceLayers() uint32        { return s.Layers }
func (s *SurfaceRgba32Float) surfaceMipmaps() uint32       { return s.Mipmaps }
func (s *SurfaceRgba32Float) get(l, d, m uint32) []float32 { return s.Get(l, d, m) }

func encodeSurface[S getMipmapSurface[P], P channel](surface S, format ImageFormat, quality Quality, mipmaps Mipmaps) (*Surface, error) {
	var numMipmaps uint32
	switch {
	case mipmaps == MipmapsDisabled:
		numMipmaps = 1
	case mipmaps == MipmapsFromSurface:
		numMipmaps = surface.surfaceMipmaps()
	case mipmaps == MipmapsGeneratedAutomatic:
		numMipmaps = maxMipmapCount(max3(surface.surfaceWidth(), surface.surfaceHeight(), surface.surfaceDepth()))
	default:
		if count, ok := mipmaps.GeneratedExactCount(); ok {
			numMipmaps = count
		} else {
			numMipmaps = 1
		}
	}

	useSurface := mipmaps == MipmapsFromSurface

	var surfaceData []byte

	for layer := uint32(0); layer < surface.surfaceLayers(); layer++ {
		// Encode 2D or 3D data for this layer.
		if err := encodeMipmapsRgba(&surfaceData, surface, format, quality, numMipmaps, useSurface, layer); err != nil {
			return nil, err
		}
	}

	return &Surface{
		Width:       surface.surfaceWidth(),
		Height:      surface.surfaceHeight(),
		Depth:       surface.surfaceDepth(),
		Layers:      surface.surfaceLayers(),
		Mipmaps:     numMipmaps,
		ImageFormat: format,
		Data:        surfaceData,
	}, nil
}

func encodeMipmapsRgba[S getMipmapSurface[P], P channel](
	surfaceData *[]byte,
	surface S,
	format ImageFormat,
	quality Quality,
	numMipmaps uint32,
	useSurface bool,
	layer uint32,
) error {
	bw, bh, bd := format.blockDimensions()
	blockDimensions := [3]uint32{bw, bh, bd}

	// Track the previous image data and dimensions.
	// This enables generating mipmaps from a single base layer.
	mipData, err := getMipmapData(surface, layer, 0, blockDimensions)
	if err != nil {
		return err
	}

	encoded, err := mipData.encode(format, quality)
	if err != nil {
		return err
	}
	*surfaceData = append(*surfaceData, encoded...)

	for mipmap := uint32(1); mipmap < numMipmaps; mipmap++ {
		if useSurface {
			mipData, err = getMipmapData(surface, layer, mipmap, blockDimensions)
			if err != nil {
				return err
			}
		} else {
			mipData = mipData.downsample(surface.surfaceWidth(), surface.surfaceHeight(), surface.surfaceDepth(), blockDimensions, mipmap)
		}

		encoded, err = mipData.encode(format, quality)
		if err != nil {
			return err
		}
		*surfaceData = append(*surfaceData, encoded...)
	}

	return nil
}

func getMipmapData[S getMipmapSurface[P], P channel](surface S, layer, mipmap uint32, blockDimensions [3]uint32) (*mipData[P], error) {
	mipWidth := MipDimension(surface.surfaceWidth(), mipmap)
	mipHeight := MipDimension(surface.surfaceHeight(), mipmap)
	mipDepth := MipDimension(surface.surfaceDepth(), mipmap)

	var data []P
	for level := uint32(0); level < surface.surfaceDepth(); level++ {
		newData := surface.get(layer, level, mipmap)
		data = append(data, newData...)
	}

	width, height, depth := physicalDimensions(mipWidth, mipHeight, mipDepth, blockDimensions)

	padded := padMipmapRgba(int(mipWidth), int(mipHeight), int(mipDepth), width, height, depth, data)

	return &mipData[P]{width: width, height: height, depth: depth, data: padded}, nil
}

// physicalDimensions rounds the dimensions up to integral dimensions in blocks.
// Applications or the GPU will use the smaller virtual size and ignore padding.
func physicalDimensions(width, height, depth uint32, blockDimensions [3]uint32) (int, int, int) {
	blockWidth, blockHeight, blockDepth := blockDimensions[0], blockDimensions[1], blockDimensions[2]
	return int(nextMultipleOf(width, blockWidth)), int(nextMultipleOf(height, blockHeight)), int(nextMultipleOf(depth, blockDepth))
}

func nextMultipleOf(value, multiple uint32) uint32 {
	if multiple <= 1 {
		return value
	}
	rem := value % multiple
	if rem == 0 {
		return value
	}
	return value + (multiple - rem)
}

// padMipmapRgba zero pads the data to the given dimensions.
func padMipmapRgba[T channel](width, height, depth, newWidth, newHeight, newDepth int, data []T) []T {
	channels := 4
	newSize := newWidth * newHeight * newDepth * channels

	if len(data) < newSize {
		// Zero pad the data to the appropriate size.
		var zero T
		paddedData := make([]T, newSize)
		for i := range paddedData {
			paddedData[i] = zero
		}
		// Copy the original data row by row.
		for z := 0; z < depth; z++ {
			for y := 0; y < height; y++ {
				// Assume padded dimensions are larger than the dimensions.
				inBase := ((z * width * height) + y*width) * channels
				outBase := ((z * newWidth * newHeight) + y*newWidth) * channels
				copy(paddedData[outBase:outBase+width*channels], data[inBase:inBase+width*channels])
			}
		}
		return paddedData
	}
	return data
}

// encodePixels encodes 2D pixel data to the given format, for u8 and f32
// channel types.
func encodePixels[T channel](width, height uint32, data []T, format ImageFormat, quality Quality) ([]byte, error) {
	switch any(*new(T)).(type) {
	case uint8:
		return encodeU8(width, height, any(data).([]uint8), format, quality)
	case float32:
		return encodeF32(width, height, any(data).([]float32), format, quality)
	}
	return nil, &SurfaceError{Kind: SurfaceErrorUnsupportedEncodeFormat, Format: format}
}

// encodeU8 encodes RGBA8 image data to the given format.
func encodeU8(width, height uint32, data []uint8, format ImageFormat, quality Quality) ([]byte, error) {
	// Unorm and srgb only affect how the data is read.
	// Use the same conversion code for both.
	switch format {
	case BC1RgbaUnorm, BC1RgbaUnormSrgb:
		return encodeBcn(bcFmt1, width, height, data, quality)
	case BC2RgbaUnorm, BC2RgbaUnormSrgb:
		return encodeBcn(bcFmt2, width, height, data, quality)
	case BC3RgbaUnorm, BC3RgbaUnormSrgb:
		return encodeBcn(bcFmt3, width, height, data, quality)
	case BC4RUnorm, BC4RSnorm:
		return encodeBcn(bcFmt4, width, height, data, quality)
	case BC5RgUnorm, BC5RgSnorm:
		return encodeBcn(bcFmt5, width, height, data, quality)
	case BC6hRgbUfloat, BC6hRgbSfloat:
		return encodeBcn(bcFmt6, width, height, data, quality)
	case BC7RgbaUnorm, BC7RgbaUnormSrgb:
		return encodeBcn(bcFmt7, width, height, data, quality)
	case R8Unorm:
		return encodeRgbaU8(formatR8, width, height, data)
	case R8Snorm:
		return encodeRgbaU8(formatR8Snorm, width, height, data)
	case Rg8Unorm:
		return encodeRgbaU8(formatRg8, width, height, data)
	case Rg8Snorm:
		return encodeRgbaU8(formatRg8Snorm, width, height, data)
	case Rgba8Unorm, Rgba8UnormSrgb:
		return encodeRgbaU8(formatRgba8, width, height, data)
	case Rgba8Snorm:
		return encodeRgbaU8(formatRgba8Snorm, width, height, data)
	case Bgra8Unorm, Bgra8UnormSrgb:
		return encodeRgbaU8(formatBgra8, width, height, data)
	case Bgra4Unorm:
		return encodeRgbaU8(formatBgra4, width, height, data)
	case Bgr8Unorm:
		return encodeRgbaU8(formatBgr8, width, height, data)
	case R16Unorm:
		return encodeRgbaU8(formatR16, width, height, data)
	case R16Snorm:
		return encodeRgbaU8(formatR16Snorm, width, height, data)
	case Rg16Unorm:
		return encodeRgbaU8(formatRg16, width, height, data)
	case Rg16Snorm:
		return encodeRgbaU8(formatRg16Snorm, width, height, data)
	case Rgba16Unorm:
		return encodeRgbaU8(formatRgba16, width, height, data)
	case Rgba16Snorm:
		return encodeRgbaU8(formatRgba16Snorm, width, height, data)
	case R16Float:
		// R16Float encoded from u8 uses Rf32.
		return encodeRgbaU8(formatRf32, width, height, data)
	case Rg16Float:
		return encodeRgbaU8(formatRgf16, width, height, data)
	case Rgba16Float:
		return encodeRgbaU8(formatRgbaf16, width, height, data)
	case R32Float:
		return encodeRgbaU8(formatRf32, width, height, data)
	case Rg32Float:
		return encodeRgbaU8(formatRgf32, width, height, data)
	case Rgb32Float:
		return encodeRgbaU8(formatRgbf32, width, height, data)
	case Rgba32Float:
		return encodeRgbaU8(formatRgbaf32, width, height, data)
	case Bgr5A1Unorm:
		return encodeRgbaU8(formatBgr5A1, width, height, data)
	}
	return nil, &SurfaceError{Kind: SurfaceErrorUnsupportedEncodeFormat, Format: format}
}

// encodeF32 encodes RGBAF32 image data to the given format.
func encodeF32(width, height uint32, data []float32, format ImageFormat, quality Quality) ([]byte, error) {
	// Unorm and srgb only affect how the data is read.
	// Use the same conversion code for both.
	switch format {
	case R8Snorm:
		return encodeRgbaF32(formatR8Snorm, width, height, data)
	case Rg8Snorm:
		return encodeRgbaF32(formatRg8Snorm, width, height, data)
	case Rgba8Snorm:
		return encodeRgbaF32(formatRgba8Snorm, width, height, data)
	case BC4RSnorm, BC5RgSnorm:
		// No dedicated encoder for snorm formats.
		rgba8 := make([]uint8, len(data))
		for i, f := range data {
			rgba8[i] = uint8(floatToSnorm8(f))
		}
		return encodeU8(width, height, rgba8, format, quality)
	case BC6hRgbUfloat, BC6hRgbSfloat:
		return encodeBcn(bcFmt6, width, height, data, quality)
	case R16Float:
		return encodeRgbaF32(formatRf16, width, height, data)
	case Rg16Float:
		return encodeRgbaF32(formatRgf16, width, height, data)
	case Rgba16Float:
		return encodeRgbaF32(formatRgbaf16, width, height, data)
	case R32Float:
		return encodeRgbaF32(formatRf32, width, height, data)
	case Rg32Float:
		return encodeRgbaF32(formatRgf32, width, height, data)
	case Rgba32Float:
		return encodeRgbaF32(formatRgbaf32, width, height, data)
	case R16Unorm:
		return encodeRgbaF32(formatR16, width, height, data)
	case Rg16Unorm:
		return encodeRgbaF32(formatRg16, width, height, data)
	case Rgba16Unorm:
		return encodeRgbaF32(formatRgba16, width, height, data)
	case R16Snorm:
		return encodeRgbaF32(formatR16Snorm, width, height, data)
	case Rg16Snorm:
		return encodeRgbaF32(formatRg16Snorm, width, height, data)
	case Rgba16Snorm:
		return encodeRgbaF32(formatRgba16Snorm, width, height, data)
	default:
		rgba8 := make([]uint8, len(data))
		for i, f := range data {
			rgba8[i] = f32ToU8(f * 255.0)
		}
		return encodeU8(width, height, rgba8, format, quality)
	}
}
