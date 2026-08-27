package ddsutil

// Quality is the conversion quality when encoding to compressed formats.
type Quality uint8

const (
	// QualityFast is faster exports with slightly lower quality.
	QualityFast Quality = iota
	// QualityNormal is normal export speed and quality.
	QualityNormal
	// QualitySlow is slower exports for slightly higher quality.
	QualitySlow
)

// Mipmaps specifies how many mipmaps to generate.
// Mipmaps are counted starting from the base level,
// so a surface with only the full resolution base level has 1 mipmap.
type Mipmaps uint32

const (
	// MipmapsDisabled means no mipmapping; only the base mip level is used.
	MipmapsDisabled Mipmaps = 0
	// MipmapsFromSurface uses the number of mipmaps specified in the input surface.
	MipmapsFromSurface Mipmaps = 1
	// MipmapsGeneratedAutomatic generates mipmaps starting from the base level
	// until dimensions can be reduced no further.
	MipmapsGeneratedAutomatic Mipmaps = 2
	// MipmapsGeneratedExact generates mipmaps to create a surface with the
	// desired number of mipmaps. A value of 0 or 1 is equivalent to MipmapsDisabled.
	MipmapsGeneratedExact Mipmaps = 3
)

// GeneratedExact returns a Mipmaps value that generates exactly count mipmaps.
func GeneratedExact(count uint32) Mipmaps { return Mipmaps(4 + count) }

// GeneratedExactCount returns the count for a GeneratedExact value, or 0.
func (m Mipmaps) GeneratedExactCount() (uint32, bool) {
	if m >= Mipmaps(4) {
		return uint32(m) - 4, true
	}
	return 0, false
}

// ImageFormat is a supported image format for encoding and decoding.
type ImageFormat uint8

const (
	R8Unorm ImageFormat = iota
	R8Snorm
	Rg8Unorm
	Rg8Snorm
	Rgba8Unorm
	Rgba8UnormSrgb
	Rgba16Float
	Rgba32Float
	Bgr8Unorm
	Bgra8Unorm
	Bgra8UnormSrgb
	Bgra4Unorm
	// DXT1
	BC1RgbaUnorm
	BC1RgbaUnormSrgb
	// DXT3
	BC2RgbaUnorm
	BC2RgbaUnormSrgb
	// DXT5
	BC3RgbaUnorm
	BC3RgbaUnormSrgb
	// RGTC1
	BC4RUnorm
	BC4RSnorm
	// RGTC2
	BC5RgUnorm
	BC5RgSnorm
	// BPTC (float)
	BC6hRgbUfloat
	BC6hRgbSfloat
	// BPTC (unorm)
	BC7RgbaUnorm
	BC7RgbaUnormSrgb
	Rgba8Snorm
	R16Unorm
	R16Snorm
	Rg16Unorm
	Rg16Snorm
	Rgba16Unorm
	Rgba16Snorm
	R16Float
	Rg16Float
	R32Float
	Rg32Float
	Rgb32Float
	Bgr5A1Unorm

	imageFormatCount
)

// String returns the name of the format.
func (f ImageFormat) String() string {
	switch f {
	case R8Unorm:
		return "R8Unorm"
	case R8Snorm:
		return "R8Snorm"
	case Rg8Unorm:
		return "Rg8Unorm"
	case Rg8Snorm:
		return "Rg8Snorm"
	case Rgba8Unorm:
		return "Rgba8Unorm"
	case Rgba8UnormSrgb:
		return "Rgba8UnormSrgb"
	case Rgba16Float:
		return "Rgba16Float"
	case Rgba32Float:
		return "Rgba32Float"
	case Bgr8Unorm:
		return "Bgr8Unorm"
	case Bgra8Unorm:
		return "Bgra8Unorm"
	case Bgra8UnormSrgb:
		return "Bgra8UnormSrgb"
	case Bgra4Unorm:
		return "Bgra4Unorm"
	case BC1RgbaUnorm:
		return "BC1RgbaUnorm"
	case BC1RgbaUnormSrgb:
		return "BC1RgbaUnormSrgb"
	case BC2RgbaUnorm:
		return "BC2RgbaUnorm"
	case BC2RgbaUnormSrgb:
		return "BC2RgbaUnormSrgb"
	case BC3RgbaUnorm:
		return "BC3RgbaUnorm"
	case BC3RgbaUnormSrgb:
		return "BC3RgbaUnormSrgb"
	case BC4RUnorm:
		return "BC4RUnorm"
	case BC4RSnorm:
		return "BC4RSnorm"
	case BC5RgUnorm:
		return "BC5RgUnorm"
	case BC5RgSnorm:
		return "BC5RgSnorm"
	case BC6hRgbUfloat:
		return "BC6hRgbUfloat"
	case BC6hRgbSfloat:
		return "BC6hRgbSfloat"
	case BC7RgbaUnorm:
		return "BC7RgbaUnorm"
	case BC7RgbaUnormSrgb:
		return "BC7RgbaUnormSrgb"
	case Rgba8Snorm:
		return "Rgba8Snorm"
	case R16Unorm:
		return "R16Unorm"
	case R16Snorm:
		return "R16Snorm"
	case Rg16Unorm:
		return "Rg16Unorm"
	case Rg16Snorm:
		return "Rg16Snorm"
	case Rgba16Unorm:
		return "Rgba16Unorm"
	case Rgba16Snorm:
		return "Rgba16Snorm"
	case R16Float:
		return "R16Float"
	case Rg16Float:
		return "Rg16Float"
	case R32Float:
		return "R32Float"
	case Rg32Float:
		return "Rg32Float"
	case Rgb32Float:
		return "Rgb32Float"
	case Bgr5A1Unorm:
		return "Bgr5A1Unorm"
	}
	return "Unknown"
}

// AllImageFormats returns all supported image formats in declaration order.
func AllImageFormats() []ImageFormat {
	formats := make([]ImageFormat, 0, imageFormatCount)
	for i := 0; i < int(imageFormatCount); i++ {
		formats = append(formats, ImageFormat(i))
	}
	return formats
}

// blockDimensions returns the block dimensions (width, height, depth) of the format.
// Uncompressed formats use 1x1x1 blocks.
func (f ImageFormat) blockDimensions() (uint32, uint32, uint32) {
	switch f {
	case BC1RgbaUnorm, BC1RgbaUnormSrgb,
		BC2RgbaUnorm, BC2RgbaUnormSrgb,
		BC3RgbaUnorm, BC3RgbaUnormSrgb,
		BC4RUnorm, BC4RSnorm,
		BC5RgUnorm, BC5RgSnorm,
		BC6hRgbUfloat, BC6hRgbSfloat,
		BC7RgbaUnorm, BC7RgbaUnormSrgb:
		return 4, 4, 1
	}
	return 1, 1, 1
}

// BlockSizeInBytes returns the size of a block if compressed, or a pixel if
// uncompressed.
func (f ImageFormat) BlockSizeInBytes() int {
	switch f {
	case R8Unorm, R8Snorm:
		return 1
	case Rg8Unorm, Rg8Snorm:
		return 2
	case Rgba8Unorm, Rgba8UnormSrgb, Rgba8Snorm, Bgra8Unorm, Bgra8UnormSrgb:
		return 4
	case Rgba16Float:
		return 8
	case Rgba32Float:
		return 16
	case Bgra4Unorm, Bgr5A1Unorm:
		return 2
	case Bgr8Unorm:
		return 3
	case R16Unorm, R16Snorm, R16Float:
		return 2
	case Rg16Unorm, Rg16Snorm, Rg16Float:
		return 4
	case Rgba16Unorm, Rgba16Snorm:
		return 8
	case Rg32Float:
		return 8
	case R32Float:
		return 4
	case Rgb32Float:
		return 12
	case BC1RgbaUnorm, BC1RgbaUnormSrgb, BC4RUnorm, BC4RSnorm:
		return 8
	case BC2RgbaUnorm, BC2RgbaUnormSrgb, BC3RgbaUnorm, BC3RgbaUnormSrgb,
		BC5RgUnorm, BC5RgSnorm, BC6hRgbUfloat, BC6hRgbSfloat,
		BC7RgbaUnorm, BC7RgbaUnormSrgb:
		return 16
	}
	return 0
}

// IsBlockCompressed returns true for the BCn formats.
func (f ImageFormat) IsBlockCompressed() bool {
	w, h, _ := f.blockDimensions()
	return w > 1 || h > 1
}
