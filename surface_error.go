package ddsutil

import "errors"

// SurfaceError enumerates errors that can occur while encoding or decoding a surface.
type SurfaceError struct {
	Kind SurfaceErrorKind
	// ZeroSizedSurface / PixelCountWouldOverflow / NonIntegralDimensionsInBlocks
	Width  uint32
	Height uint32
	Depth  uint32
	// NonIntegralDimensionsInBlocks
	BlockWidth  uint32
	BlockHeight uint32
	// NotEnoughData
	Expected int
	Actual   int
	// UnsupportedEncodeFormat / UnsupportedDdsFormat
	Format ImageFormat
	// InvalidMipmapCount / UnexpectedMipmapCount
	Mipmaps         uint32
	MaxTotalMipmaps uint32
	// MipmapDataOutOfBounds
	Layer  uint32
	Mipmap uint32
	// UnsupportedDdsFormat
	DdsFormat *DdsFormatInfo
}

type SurfaceErrorKind uint8

const (
	SurfaceErrorZeroSizedSurface SurfaceErrorKind = iota
	SurfaceErrorPixelCountWouldOverflow
	SurfaceErrorNonIntegralDimensionsInBlocks
	SurfaceErrorNotEnoughData
	SurfaceErrorUnsupportedEncodeFormat
	SurfaceErrorInvalidMipmapCount
	SurfaceErrorMipmapDataOutOfBounds
	SurfaceErrorUnsupportedDdsFormat
	SurfaceErrorUnexpectedMipmapCount
)

func (e *SurfaceError) Error() string {
	switch e.Kind {
	case SurfaceErrorZeroSizedSurface:
		return "surface dimensions contain no pixels"
	case SurfaceErrorPixelCountWouldOverflow:
		return "surface pixel count would overflow"
	case SurfaceErrorNonIntegralDimensionsInBlocks:
		return "surface dimensions are not divisible by the block dimensions"
	case SurfaceErrorNotEnoughData:
		return "expected surface to have at least " + itoa(e.Expected) + " bytes but found " + itoa(e.Actual)
	case SurfaceErrorUnsupportedEncodeFormat:
		return "encoding data to format " + e.Format.String() + " is not supported"
	case SurfaceErrorInvalidMipmapCount:
		return "mipmap count exceeds the maximum value"
	case SurfaceErrorMipmapDataOutOfBounds:
		return "failed to get image data for layer " + itoa(int(e.Layer)) + " mipmap " + itoa(int(e.Mipmap))
	case SurfaceErrorUnsupportedDdsFormat:
		return "DDS image format is not supported"
	case SurfaceErrorUnexpectedMipmapCount:
		return "mipmaps exceeds the maximum expected mipmap count"
	}
	return "unknown surface error"
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// CreateImageError enumerates errors that can occur while creating a decoded image.
type CreateImageError struct {
	Kind CreateImageErrorKind
	// InvalidSurfaceDimensions
	Width      uint32
	Height     uint32
	DataLength int
	// DecompressSurface
	SurfaceErr *SurfaceError
	// UnexpectedMipmapCount
	Mipmaps    uint32
	MaxMipmaps uint32
}

type CreateImageErrorKind uint8

const (
	CreateImageErrorInvalidSurfaceDimensions CreateImageErrorKind = iota
	CreateImageErrorDecompressSurface
	CreateImageErrorUnexpectedMipmapCount
)

func (e *CreateImageError) Error() string {
	switch e.Kind {
	case CreateImageErrorInvalidSurfaceDimensions:
		return "data length " + itoa(e.DataLength) + " is not valid for a " + itoa(int(e.Width)) + "x" + itoa(int(e.Height)) + " image"
	case CreateImageErrorDecompressSurface:
		if e.SurfaceErr != nil {
			return "error decompressing surface: " + e.SurfaceErr.Error()
		}
		return "error decompressing surface"
	case CreateImageErrorUnexpectedMipmapCount:
		return itoa(int(e.Mipmaps)) + " mipmaps exceeds the maximum expected mipmap count of " + itoa(int(e.MaxMipmaps))
	}
	return "unknown image creation error"
}

// CreateDdsError enumerates errors that can occur when converting to DDS.
type CreateDdsError struct {
	Kind CreateDdsErrorKind
	Err  error
}

type CreateDdsErrorKind uint8

const (
	CreateDdsErrorDds CreateDdsErrorKind = iota
	CreateDdsErrorCompressSurface
)

func (e *CreateDdsError) Error() string {
	switch e.Kind {
	case CreateDdsErrorDds:
		return "error creating DDS: " + errString(e.Err)
	case CreateDdsErrorCompressSurface:
		return "error compressing surface: " + errString(e.Err)
	}
	return "unknown dds creation error"
}

func (e *CreateDdsError) Unwrap() error { return e.Err }

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

var errOutOfBounds = errors.New("out of bounds")
