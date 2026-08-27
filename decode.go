package ddsutil

// DecodeRgba8 decodes all layers and mipmaps from the surface to RGBA8.
func (s *Surface) DecodeRgba8() (*SurfaceRgba8, error) {
	return s.DecodeLayersMipmapsRgba8(0, s.Layers, 0, s.Mipmaps)
}

// DecodeLayersMipmapsRgba8 decodes a range of layers and mipmaps from the
// surface to RGBA8.
func (s *Surface) DecodeLayersMipmapsRgba8(layerStart, layerEnd, mipmapStart, mipmapEnd uint32) (*SurfaceRgba8, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}

	data, err := decodeSurfaceU8(s, layerStart, layerEnd, mipmapStart, mipmapEnd)
	if err != nil {
		return nil, err
	}

	layers := layerEnd - layerStart
	if layers < 1 {
		layers = 1
	}
	mipmaps := mipmapEnd - mipmapStart
	if mipmaps < 1 {
		mipmaps = 1
	}

	return &SurfaceRgba8{
		Width:   MipDimension(s.Width, mipmapStart),
		Height:  MipDimension(s.Height, mipmapStart),
		Depth:   MipDimension(s.Depth, mipmapStart),
		Layers:  layers,
		Mipmaps: mipmaps,
		Data:    data,
	}, nil
}

// DecodeRgbaf32 decodes all layers and mipmaps from the surface to RGBAF32.
//
// Non floating point formats are normalized to the range 0.0 to 1.0.
func (s *Surface) DecodeRgbaf32() (*SurfaceRgba32Float, error) {
	return s.DecodeLayersMipmapsRgbaf32(0, s.Layers, 0, s.Mipmaps)
}

// DecodeLayersMipmapsRgbaf32 decodes a range of layers and mipmaps from the
// surface to RGBAF32.
//
// Non floating point formats are normalized to the range 0.0 to 1.0.
func (s *Surface) DecodeLayersMipmapsRgbaf32(layerStart, layerEnd, mipmapStart, mipmapEnd uint32) (*SurfaceRgba32Float, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}

	data, err := decodeSurfaceF32(s, layerStart, layerEnd, mipmapStart, mipmapEnd)
	if err != nil {
		return nil, err
	}

	layers := layerEnd - layerStart
	if layers < 1 {
		layers = 1
	}
	mipmaps := mipmapEnd - mipmapStart
	if mipmaps < 1 {
		mipmaps = 1
	}

	return &SurfaceRgba32Float{
		Width:   MipDimension(s.Width, mipmapStart),
		Height:  MipDimension(s.Height, mipmapStart),
		Depth:   MipDimension(s.Depth, mipmapStart),
		Layers:  layers,
		Mipmaps: mipmaps,
		Data:    data,
	}, nil
}

// decodeSurfaceU8 decodes the given layers and mipmaps to RGBA8 data.
func decodeSurfaceU8(surface *Surface, layerStart, layerEnd, mipmapStart, mipmapEnd uint32) ([]uint8, error) {
	var combined []uint8
	for layer := layerStart; layer < layerEnd; layer++ {
		for level := uint32(0); level < surface.Depth; level++ {
			for mipmap := mipmapStart; mipmap < mipmapEnd; mipmap++ {
				data := surface.Get(layer, level, mipmap)
				if data == nil {
					return nil, &SurfaceError{Kind: SurfaceErrorMipmapDataOutOfBounds, Layer: layer, Mipmap: mipmap}
				}

				// The mipmap index is already validated by Get above.
				width := MipDimension(surface.Width, mipmap)
				height := MipDimension(surface.Height, mipmap)

				decoded, err := decodeU8(width, height, surface.ImageFormat, data)
				if err != nil {
					return nil, err
				}

				combined = append(combined, decoded...)
			}
		}
	}
	return combined, nil
}

// decodeSurfaceF32 decodes the given layers and mipmaps to RGBAF32 data.
func decodeSurfaceF32(surface *Surface, layerStart, layerEnd, mipmapStart, mipmapEnd uint32) ([]float32, error) {
	var combined []float32
	for layer := layerStart; layer < layerEnd; layer++ {
		for level := uint32(0); level < surface.Depth; level++ {
			for mipmap := mipmapStart; mipmap < mipmapEnd; mipmap++ {
				data := surface.Get(layer, level, mipmap)
				if data == nil {
					return nil, &SurfaceError{Kind: SurfaceErrorMipmapDataOutOfBounds, Layer: layer, Mipmap: mipmap}
				}

				width := MipDimension(surface.Width, mipmap)
				height := MipDimension(surface.Height, mipmap)

				decoded, err := decodeF32(width, height, surface.ImageFormat, data)
				if err != nil {
					return nil, err
				}

				combined = append(combined, decoded...)
			}
		}
	}
	return combined, nil
}

// decodeU8 decodes 2D image data to RGBA8.
func decodeU8(width, height uint32, imageFormat ImageFormat, data []byte) ([]uint8, error) {
	switch imageFormat {
	case BC1RgbaUnorm, BC1RgbaUnormSrgb:
		return decodeBcn(bcFmt1, width, height, data)
	case BC2RgbaUnorm, BC2RgbaUnormSrgb:
		return decodeBcn(bcFmt2, width, height, data)
	case BC3RgbaUnorm, BC3RgbaUnormSrgb:
		return decodeBcn(bcFmt3, width, height, data)
	case BC4RUnorm:
		return decodeBcn(bcFmt4, width, height, data)
	case BC4RSnorm:
		return decodeBcn(bcFmt4S, width, height, data)
	case BC5RgUnorm:
		return decodeBcn(bcFmt5, width, height, data)
	case BC5RgSnorm:
		return decodeBcn(bcFmt5S, width, height, data)
	case BC6hRgbUfloat, BC6hRgbSfloat:
		return decodeBcn(bcFmt6, width, height, data)
	case BC7RgbaUnorm, BC7RgbaUnormSrgb:
		return decodeBcn(bcFmt7, width, height, data)
	case R8Unorm:
		return decodeRgbaU8(formatR8, width, height, data)
	case R8Snorm:
		return decodeRgbaU8(formatR8Snorm, width, height, data)
	case Rg8Unorm:
		return decodeRgbaU8(formatRg8, width, height, data)
	case Rg8Snorm:
		return decodeRgbaU8(formatRg8Snorm, width, height, data)
	case Rgba8Unorm, Rgba8UnormSrgb:
		return decodeRgbaU8(formatRgba8, width, height, data)
	case Rgba16Float:
		return decodeRgbaU8(formatRgbaf16, width, height, data)
	case Rgba32Float:
		return decodeRgbaU8(formatRgbaf32, width, height, data)
	case Bgra8Unorm, Bgra8UnormSrgb:
		return decodeRgbaU8(formatBgra8, width, height, data)
	case Rgba8Snorm:
		return decodeRgbaU8(formatRgba8Snorm, width, height, data)
	case Bgra4Unorm:
		return decodeRgbaU8(formatBgra4, width, height, data)
	case Bgr8Unorm:
		return decodeRgbaU8(formatBgr8, width, height, data)
	case R16Unorm:
		return decodeRgbaU8(formatR16, width, height, data)
	case R16Snorm:
		return decodeRgbaU8(formatR16Snorm, width, height, data)
	case Rg16Unorm:
		return decodeRgbaU8(formatRg16, width, height, data)
	case Rg16Snorm:
		return decodeRgbaU8(formatRg16Snorm, width, height, data)
	case Rgba16Unorm:
		return decodeRgbaU8(formatRgba16, width, height, data)
	case Rgba16Snorm:
		return decodeRgbaU8(formatRgba16Snorm, width, height, data)
	case Rg16Float:
		return decodeRgbaU8(formatRgf16, width, height, data)
	case Rg32Float:
		return decodeRgbaU8(formatRgf32, width, height, data)
	case R16Float:
		return decodeRgbaU8(formatRf16, width, height, data)
	case R32Float:
		return decodeRgbaU8(formatRf32, width, height, data)
	case Rgb32Float:
		return decodeRgbaU8(formatRgbf32, width, height, data)
	case Bgr5A1Unorm:
		return decodeRgbaU8(formatBgr5A1, width, height, data)
	}
	return nil, &SurfaceError{Kind: SurfaceErrorUnsupportedEncodeFormat, Format: imageFormat}
}

// decodeF32 decodes 2D image data to RGBAF32.
func decodeF32(width, height uint32, imageFormat ImageFormat, data []byte) ([]float32, error) {
	switch imageFormat {
	case R8Snorm:
		return decodeRgbaF32(formatR8Snorm, width, height, data)
	case Rg8Snorm:
		return decodeRgbaF32(formatRg8Snorm, width, height, data)
	case Rgba8Snorm:
		return decodeRgbaF32(formatRgba8Snorm, width, height, data)
	case BC4RSnorm:
		return decodeBcnF32(bcFmt4S, width, height, data)
	case BC5RgSnorm:
		return decodeBcnF32(bcFmt5S, width, height, data)
	case BC6hRgbUfloat, BC6hRgbSfloat:
		return decodeBcnF32(bcFmt6, width, height, data)
	case R16Float:
		return decodeRgbaF32(formatRf16, width, height, data)
	case Rg16Float:
		return decodeRgbaF32(formatRgf16, width, height, data)
	case Rgba16Float:
		return decodeRgbaF32(formatRgbaf16, width, height, data)
	case R32Float:
		return decodeRgbaF32(formatRf32, width, height, data)
	case Rg32Float:
		return decodeRgbaF32(formatRgf32, width, height, data)
	case Rgb32Float:
		return decodeRgbaF32(formatRgbf32, width, height, data)
	case Rgba32Float:
		return decodeRgbaF32(formatRgbaf32, width, height, data)
	case R16Unorm:
		return decodeRgbaF32(formatR16, width, height, data)
	case Rg16Unorm:
		return decodeRgbaF32(formatRg16, width, height, data)
	case Rgba16Unorm:
		return decodeRgbaF32(formatRgba16, width, height, data)
	case R16Snorm:
		return decodeRgbaF32(formatR16Snorm, width, height, data)
	case Rg16Snorm:
		return decodeRgbaF32(formatRg16Snorm, width, height, data)
	case Rgba16Snorm:
		return decodeRgbaF32(formatRgba16Snorm, width, height, data)
	default:
		// Use existing decoding for formats that don't store floating point data.
		rgba8, err := decodeU8(width, height, imageFormat, data)
		if err != nil {
			return nil, err
		}
		out := make([]float32, len(rgba8))
		for i, u := range rgba8 {
			out[i] = float32(u) / 255.0
		}
		return out, nil
	}
}
