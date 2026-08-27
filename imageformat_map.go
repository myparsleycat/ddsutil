package ddsutil

// This file is the bridge between the file container layer (the Dds type and
// the D3DFormat/DxgiFormat enums) and the image codec layer (the Surface
// types and the ImageFormat enum).
//
// All header parsing, format enums, and pitch/stride computation come from
// the container layer.

import "bytes"

// DdsFormatInfo is the format information for all DDS variants.
type DdsFormatInfo struct {
	DXGI   *DxgiFormat
	D3D    *D3DFormat
	FourCC *FourCC
}

// fourccBC5U is the LE encoding of "BC5U".
const fourccBC5U FourCC = 0x55354342

// Parse parses a DDS file from an in-memory byte slice. It is a convenience
// wrapper around Read.
func Parse(data []byte) (*Dds, error) {
	return Read(bytes.NewReader(data))
}

// Bytes serializes the DDS file to a byte slice.
func (d *Dds) Bytes() ([]byte, error) {
	var buf bytes.Buffer
	if err := d.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DDSImageFormat returns the ImageFormat of the DDS, or an error describing
// the unrecognized format.
func DDSImageFormat(dds *Dds) (ImageFormat, error) {
	// Legacy D3D formats first: the DXTn and float FourCC codes, plus the
	// RGB bitmask layouts, are all detected by the container layer.
	if d3d, ok := dds.GetD3DFormat(); ok {
		if f, ok := imageFormatFromD3D(d3d); ok {
			return f, nil
		}
	}
	// DX10 extension header carries the DXGI format.
	if dds.Header10 != nil {
		if f, ok := imageFormatFromDXGI(dds.Header10.DxgiFormat); ok {
			return f, nil
		}
	}
	// FourCC-only codes (BC4U/BC4S/ATI2/BC5U/BC5S) that are not part of the
	// legacy D3D format table.
	if dds.Header.SPF.FourCC != nil {
		if f, ok := imageFormatFromFourCC(*dds.Header.SPF.FourCC); ok {
			return f, nil
		}
	}
	return 0, &SurfaceError{
		Kind:      SurfaceErrorUnsupportedDdsFormat,
		DdsFormat: ddsFormatInfo(dds),
	}
}

func ddsFormatInfo(dds *Dds) *DdsFormatInfo {
	info := &DdsFormatInfo{}
	if dds.Header10 != nil {
		f := dds.Header10.DxgiFormat
		info.DXGI = &f
	}
	if f, ok := dds.GetD3DFormat(); ok {
		info.D3D = &f
	}
	if dds.Header.SPF.FourCC != nil {
		f := *dds.Header.SPF.FourCC
		info.FourCC = &f
	}
	return info
}

func imageFormatFromDXGI(format DxgiFormat) (ImageFormat, bool) {
	switch format {
	case DxgiFormatR8_UNorm:
		return R8Unorm, true
	case DxgiFormatR8_SNorm:
		return R8Snorm, true
	case DxgiFormatR8G8_UNorm:
		return Rg8Unorm, true
	case DxgiFormatR8G8_SNorm:
		return Rg8Snorm, true
	case DxgiFormatR8G8B8A8_UNorm:
		return Rgba8Unorm, true
	case DxgiFormatR8G8B8A8_UNorm_sRGB:
		return Rgba8UnormSrgb, true
	case DxgiFormatR16G16B16A16_Float:
		return Rgba16Float, true
	case DxgiFormatR32G32B32A32_Float:
		return Rgba32Float, true
	case DxgiFormatB8G8R8A8_UNorm:
		return Bgra8Unorm, true
	case DxgiFormatB8G8R8A8_UNorm_sRGB:
		return Bgra8UnormSrgb, true
	case DxgiFormatBC1_UNorm:
		return BC1RgbaUnorm, true
	case DxgiFormatBC1_UNorm_sRGB:
		return BC1RgbaUnormSrgb, true
	case DxgiFormatBC2_UNorm:
		return BC2RgbaUnorm, true
	case DxgiFormatBC2_UNorm_sRGB:
		return BC2RgbaUnormSrgb, true
	case DxgiFormatBC3_UNorm:
		return BC3RgbaUnorm, true
	case DxgiFormatBC3_UNorm_sRGB:
		return BC3RgbaUnormSrgb, true
	case DxgiFormatBC4_UNorm:
		return BC4RUnorm, true
	case DxgiFormatBC4_SNorm:
		return BC4RSnorm, true
	case DxgiFormatBC5_UNorm:
		return BC5RgUnorm, true
	case DxgiFormatBC5_SNorm:
		return BC5RgSnorm, true
	case DxgiFormatBC6H_SF16:
		return BC6hRgbSfloat, true
	case DxgiFormatBC6H_UF16:
		return BC6hRgbUfloat, true
	case DxgiFormatBC7_UNorm:
		return BC7RgbaUnorm, true
	case DxgiFormatBC7_UNorm_sRGB:
		return BC7RgbaUnormSrgb, true
	case DxgiFormatB4G4R4A4_UNorm:
		return Bgra4Unorm, true
	case DxgiFormatR16G16B16A16_UNorm:
		return Rgba16Unorm, true
	case DxgiFormatR16G16B16A16_SNorm:
		return Rgba16Snorm, true
	case DxgiFormatR16G16_UNorm:
		return Rg16Unorm, true
	case DxgiFormatR16G16_SNorm:
		return Rg16Snorm, true
	case DxgiFormatR16_UNorm:
		return R16Unorm, true
	case DxgiFormatR16_SNorm:
		return R16Snorm, true
	case DxgiFormatR16_Float:
		return R16Float, true
	case DxgiFormatR16G16_Float:
		return Rg16Float, true
	case DxgiFormatR32_Float:
		return R32Float, true
	case DxgiFormatR32G32_Float:
		return Rg32Float, true
	case DxgiFormatR8G8B8A8_SNorm:
		return Rgba8Snorm, true
	case DxgiFormatR32G32B32_Float:
		return Rgb32Float, true
	case DxgiFormatB5G5R5A1_UNorm:
		return Bgr5A1Unorm, true
	}
	return 0, false
}

func imageFormatFromD3D(format D3DFormat) (ImageFormat, bool) {
	switch format {
	case D3DFormatDXT1:
		return BC1RgbaUnorm, true
	case D3DFormatDXT2, D3DFormatDXT3:
		return BC2RgbaUnorm, true
	case D3DFormatDXT4, D3DFormatDXT5:
		return BC3RgbaUnorm, true
	// BGRA can also be written ARGB depending on how we look at the bytes.
	case D3DFormatA4R4G4B4:
		return Bgra4Unorm, true
	case D3DFormatA8R8G8B8:
		return Bgra8Unorm, true
	case D3DFormatR8G8B8:
		return Bgr8Unorm, true
	case D3DFormatA8B8G8R8:
		return Rgba8Unorm, true
	case D3DFormatG16R16F:
		return Rg16Float, true
	case D3DFormatA16B16G16R16F:
		return Rgba16Float, true
	case D3DFormatG32R32F:
		return Rg32Float, true
	case D3DFormatA32B32G32R32F:
		return Rgba32Float, true
	case D3DFormatG16R16:
		return Rg16Unorm, true
	case D3DFormatA16B16G16R16:
		return Rgba16Unorm, true
	case D3DFormatA1R5G5B5:
		return Bgr5A1Unorm, true
	}
	return 0, false
}

func imageFormatFromFourCC(fourcc FourCC) (ImageFormat, bool) {
	switch fourcc {
	case FourCCDXT1:
		return BC1RgbaUnorm, true
	case FourCCDXT2, FourCCDXT3:
		return BC2RgbaUnorm, true
	case FourCCDXT4, FourCCDXT5:
		return BC3RgbaUnorm, true
	case FourCCBC4_UNORM:
		return BC4RUnorm, true
	case FourCCBC4_SNORM:
		return BC4RSnorm, true
	case FourCCATI2, fourccBC5U:
		return BC5RgUnorm, true
	case FourCCBC5_SNORM:
		return BC5RgSnorm, true
	}
	return 0, false
}

func d3DFromImageFormat(value ImageFormat) (D3DFormat, bool) {
	// bc4 and bc5 are handled by fourcc.
	switch value {
	case BC1RgbaUnorm, BC1RgbaUnormSrgb:
		return D3DFormatDXT1, true
	case BC2RgbaUnorm, BC2RgbaUnormSrgb:
		return D3DFormatDXT2, true
	case BC3RgbaUnorm, BC3RgbaUnormSrgb:
		return D3DFormatDXT5, true
	case Rgba8Unorm, Rgba8UnormSrgb:
		return D3DFormatA8B8G8R8, true
	case Rgba16Float:
		return D3DFormatA16B16G16R16F, true
	case Rgba32Float:
		return D3DFormatA32B32G32R32F, true
	case Bgra8Unorm, Bgra8UnormSrgb:
		return D3DFormatA8R8G8B8, true
	case Bgra4Unorm:
		return D3DFormatA4R4G4B4, true
	case Bgr8Unorm:
		return D3DFormatR8G8B8, true
	case Rg16Unorm:
		return D3DFormatG16R16, true
	case Rgba16Unorm:
		return D3DFormatA16B16G16R16, true
	case Rg16Float:
		return D3DFormatG16R16F, true
	case Rg32Float:
		return D3DFormatG32R32F, true
	case R16Float:
		return D3DFormatR16F, true
	case R32Float:
		return D3DFormatR32F, true
	case Bgr5A1Unorm:
		return D3DFormatA1R5G5B5, true
	}
	return 0, false
}

func dxgiFromImageFormat(value ImageFormat) (DxgiFormat, bool) {
	switch value {
	case BC1RgbaUnorm:
		return DxgiFormatBC1_UNorm, true
	case BC1RgbaUnormSrgb:
		return DxgiFormatBC1_UNorm_sRGB, true
	case BC2RgbaUnorm:
		return DxgiFormatBC2_UNorm, true
	case BC2RgbaUnormSrgb:
		return DxgiFormatBC2_UNorm_sRGB, true
	case BC3RgbaUnorm:
		return DxgiFormatBC3_UNorm, true
	case BC3RgbaUnormSrgb:
		return DxgiFormatBC3_UNorm_sRGB, true
	case BC4RUnorm:
		return DxgiFormatBC4_UNorm, true
	case BC4RSnorm:
		return DxgiFormatBC4_SNorm, true
	case BC5RgUnorm:
		return DxgiFormatBC5_UNorm, true
	case BC5RgSnorm:
		return DxgiFormatBC5_SNorm, true
	case BC6hRgbUfloat:
		return DxgiFormatBC6H_UF16, true
	case BC6hRgbSfloat:
		return DxgiFormatBC6H_SF16, true
	case BC7RgbaUnorm:
		return DxgiFormatBC7_UNorm, true
	case BC7RgbaUnormSrgb:
		return DxgiFormatBC7_UNorm_sRGB, true
	case R8Unorm:
		return DxgiFormatR8_UNorm, true
	case R8Snorm:
		return DxgiFormatR8_SNorm, true
	case Rg8Unorm:
		return DxgiFormatR8G8_UNorm, true
	case Rg8Snorm:
		return DxgiFormatR8G8_SNorm, true
	case Rgba8Unorm:
		return DxgiFormatR8G8B8A8_UNorm, true
	case Rgba8UnormSrgb:
		return DxgiFormatR8G8B8A8_UNorm_sRGB, true
	case Rgba16Float:
		return DxgiFormatR16G16B16A16_Float, true
	case Rgba32Float:
		return DxgiFormatR32G32B32A32_Float, true
	case Bgra8Unorm:
		return DxgiFormatB8G8R8A8_UNorm, true
	case Bgra8UnormSrgb:
		return DxgiFormatB8G8R8A8_UNorm_sRGB, true
	case Bgra4Unorm:
		return DxgiFormatB4G4R4A4_UNorm, true
	case R16Unorm:
		return DxgiFormatR16_UNorm, true
	case R16Snorm:
		return DxgiFormatR16_SNorm, true
	case Rg16Unorm:
		return DxgiFormatR16G16_UNorm, true
	case Rg16Snorm:
		return DxgiFormatR16G16_SNorm, true
	case Rgba16Unorm:
		return DxgiFormatR16G16B16A16_UNorm, true
	case Rgba16Snorm:
		return DxgiFormatR16G16B16A16_SNorm, true
	case Rg16Float:
		return DxgiFormatR16G16_Float, true
	case Rg32Float:
		return DxgiFormatR32G32_Float, true
	case R16Float:
		return DxgiFormatR16_Float, true
	case R32Float:
		return DxgiFormatR32_Float, true
	case Rgba8Snorm:
		return DxgiFormatR8G8B8A8_SNorm, true
	case Rgb32Float:
		return DxgiFormatR32G32B32_Float, true
	case Bgr5A1Unorm:
		return DxgiFormatB5G5R5A1_UNorm, true
	}
	return 0, false
}

// ToDds creates a DDS file with the same image data and format.
//
// Creates a DXGI DDS for most formats and D3D DDS for some legacy formats.
func (s *Surface) ToDds() (*Dds, error) {
	var dds *Dds

	if format, ok := dxgiFromImageFormat(s.ImageFormat); ok {
		var depth *uint32
		if s.Depth > 1 {
			d := s.Depth
			depth = &d
		}
		var mipmapLevels *uint32
		if s.Mipmaps > 1 {
			m := s.Mipmaps
			mipmapLevels = &m
		}
		var arrayLayers *uint32
		if s.Layers > 1 && s.Layers != 6 {
			l := s.Layers
			arrayLayers = &l
		}
		var caps2 Caps2
		if s.Layers == 6 {
			caps2 = Caps2CUBEMAP | Caps2CUBEMAP_ALLFACES
		}
		resourceDimension := D3D10ResourceDimensionTexture2D
		if s.Depth > 1 {
			resourceDimension = D3D10ResourceDimensionTexture3D
		}
		d, err := NewDXGI(NewDxgiParams{
			Height:            s.Height,
			Width:             s.Width,
			Depth:             depth,
			Format:            format,
			MipmapLevels:      mipmapLevels,
			ArrayLayers:       arrayLayers,
			Caps2:             &caps2,
			IsCubemap:         s.Layers == 6,
			ResourceDimension: resourceDimension,
			AlphaMode:         AlphaModeStraight,
		})
		if err != nil {
			return nil, &CreateDdsError{Kind: CreateDdsErrorDds, Err: err}
		}
		dds = d
	} else {
		// Not all surface formats are supported by DXGI.
		format, ok := d3DFromImageFormat(s.ImageFormat)
		if !ok {
			return nil, &CreateDdsError{Kind: CreateDdsErrorCompressSurface, Err: &SurfaceError{Kind: SurfaceErrorUnsupportedEncodeFormat, Format: s.ImageFormat}}
		}
		var depth *uint32
		if s.Depth > 1 {
			d := s.Depth
			depth = &d
		}
		var mipmapLevels *uint32
		if s.Mipmaps > 1 {
			m := s.Mipmaps
			mipmapLevels = &m
		}
		var caps2 Caps2
		if s.Layers == 6 {
			caps2 = Caps2CUBEMAP | Caps2CUBEMAP_ALLFACES
		}
		d, err := NewD3D(NewD3dParams{
			Height:       s.Height,
			Width:        s.Width,
			Depth:        depth,
			Format:       format,
			MipmapLevels: mipmapLevels,
			Caps2:        &caps2,
		})
		if err != nil {
			return nil, &CreateDdsError{Kind: CreateDdsErrorDds, Err: err}
		}
		dds = d
	}

	dds.Data = append([]byte(nil), s.Data...)

	return dds, nil
}

// SurfaceFromDds creates a view over the data in dds without any copies.
func SurfaceFromDds(dds *Dds) (*Surface, error) {
	width := dds.GetWidth()
	height := dds.GetHeight()
	depth := dds.GetDepth()
	layers := arrayLayerCount(dds)
	mipmaps := dds.GetNumMipmapLevels()
	imageFormat, err := DDSImageFormat(dds)
	if err != nil {
		return nil, err
	}

	return &Surface{
		Width:       width,
		Height:      height,
		Depth:       depth,
		Layers:      layers,
		Mipmaps:     mipmaps,
		ImageFormat: imageFormat,
		Data:        dds.Data,
	}, nil
}

func arrayLayerCount(dds *Dds) uint32 {
	// Array layers for DDS are calculated differently for cube maps.
	if dds.Header10 != nil && dds.Header10.MiscFlag.Contains(MiscFlagTEXTURECUBE) {
		n := dds.GetNumArrayLayers()
		if n < 1 {
			n = 1
		}
		return n * 6
	}
	n := dds.GetNumArrayLayers()
	if n < 1 {
		n = 1
	}
	return n
}

// DecodeDds decodes all layers and mipmaps from the DDS to an RGBA8 surface.
func DecodeDds(dds *Dds) (*SurfaceRgba8, error) {
	surface, err := SurfaceFromDds(dds)
	if err != nil {
		return nil, err
	}
	return surface.DecodeRgba8()
}

// DecodeLayersMipmapsDds decodes a specific range of layers and mipmaps from
// the DDS to an RGBA8 surface.
func DecodeLayersMipmapsDds(dds *Dds, layerStart, layerEnd, mipmapStart, mipmapEnd uint32) (*SurfaceRgba8, error) {
	surface, err := SurfaceFromDds(dds)
	if err != nil {
		return nil, err
	}
	return surface.DecodeLayersMipmapsRgba8(layerStart, layerEnd, mipmapStart, mipmapEnd)
}

// DecodeDdsF32 decodes all layers and mipmaps from the DDS to an RGBAF32 surface.
func DecodeDdsF32(dds *Dds) (*SurfaceRgba32Float, error) {
	surface, err := SurfaceFromDds(dds)
	if err != nil {
		return nil, err
	}
	return surface.DecodeRgbaf32()
}

// DecodeLayersMipmapsDdsF32 decodes a specific range of layers and mipmaps
// from the DDS to an RGBAF32 surface.
func DecodeLayersMipmapsDdsF32(dds *Dds, layerStart, layerEnd, mipmapStart, mipmapEnd uint32) (*SurfaceRgba32Float, error) {
	surface, err := SurfaceFromDds(dds)
	if err != nil {
		return nil, err
	}
	return surface.DecodeLayersMipmapsRgbaf32(layerStart, layerEnd, mipmapStart, mipmapEnd)
}

// EncodeDds encodes an RGBA8 surface to a DDS file with the given format.
// The number of mipmaps generated depends on the mipmaps parameter.
func (s *SurfaceRgba8) EncodeDds(format ImageFormat, quality Quality, mipmaps Mipmaps) (*Dds, error) {
	surface, err := s.Encode(format, quality, mipmaps)
	if err != nil {
		return nil, &CreateDdsError{Kind: CreateDdsErrorCompressSurface, Err: err}
	}
	return surface.ToDds()
}
