package ddsutil

import (
	"errors"
	"image"
)

// ImageFromDds decodes the given mip level from the DDS to an RGBA image.
// Array layers are arranged vertically from top to bottom.
//
// The pixels hold the straight (non-premultiplied) alpha stored in the DDS,
// whereas image.RGBA is defined as alpha-premultiplied. Use NRGBAFromDds
// when the image has alpha and will be composited or encoded, e.g. as PNG.
func ImageFromDds(dds *Dds, mipmap uint32) (*image.RGBA, error) {
	layers := arrayLayerCount(dds)
	surface, err := DecodeLayersMipmapsDds(dds, 0, layers, mipmap, mipmap+1)
	if err != nil {
		var surfaceErr *SurfaceError
		if !errors.As(err, &surfaceErr) {
			surfaceErr = nil
		}
		return nil, &CreateImageError{Kind: CreateImageErrorDecompressSurface, SurfaceErr: surfaceErr}
	}
	return surface.IntoImage()
}

// NRGBAFromDds decodes the given mip level from the DDS to an NRGBA image,
// which matches the straight alpha stored in DDS files. Array layers are
// arranged vertically from top to bottom.
func NRGBAFromDds(dds *Dds, mipmap uint32) (*image.NRGBA, error) {
	img, err := ImageFromDds(dds, mipmap)
	if err != nil {
		return nil, err
	}
	return &image.NRGBA{Pix: img.Pix, Stride: img.Stride, Rect: img.Rect}, nil
}

// DdsFromImage encodes image to a 2D DDS file with the given format.
// The number of mipmaps generated depends on the mipmaps parameter.
func DdsFromImage(img *image.RGBA, format ImageFormat, quality Quality, mipmaps Mipmaps) (*Dds, error) {
	// Assume all images are 2D for now.
	surface := SurfaceRgba8FromImage(img)
	encoded, err := surface.Encode(format, quality, mipmaps)
	if err != nil {
		return nil, &CreateDdsError{Kind: CreateDdsErrorCompressSurface, Err: err}
	}
	return encoded.ToDds()
}

// SurfaceRgba8FromImage creates a 2D surface from the pixels of img. It is a
// view without copies when the rows of img are contiguous, as for images
// created with image.NewRGBA, and a copy otherwise (e.g. for sub-images).
func SurfaceRgba8FromImage(img *image.RGBA) *SurfaceRgba8 {
	return SurfaceRgba8FromImageLayers(img, 1)
}

// SurfaceRgba8FromImageLayers creates a 2D surface with layers from the
// pixels of img, without copies when possible (see SurfaceRgba8FromImage).
//
// Array layers should be stacked vertically in img with an overall height height*layers.
func SurfaceRgba8FromImageLayers(img *image.RGBA, layers uint32) *SurfaceRgba8 {
	return &SurfaceRgba8{
		Width:   uint32(img.Rect.Dx()),
		Height:  uint32(img.Rect.Dy()) / layers,
		Depth:   1,
		Layers:  layers,
		Mipmaps: 1,
		Data:    contiguousPix(img),
	}
}

// SurfaceRgba8FromImageDepth creates a 3D surface from the pixels of img,
// without copies when possible (see SurfaceRgba8FromImage).
//
// Depth slices should be stacked vertically in img with an overall height height*depth.
func SurfaceRgba8FromImageDepth(img *image.RGBA, depth uint32) *SurfaceRgba8 {
	return &SurfaceRgba8{
		Width:   uint32(img.Rect.Dx()),
		Height:  uint32(img.Rect.Dy()) / depth,
		Depth:   depth,
		Layers:  1,
		Mipmaps: 1,
		Data:    contiguousPix(img),
	}
}

// contiguousPix returns the pixels of img as tightly packed rows, copying
// only if the rows are not contiguous in img.Pix. If img.Pix is too short for
// img.Rect, the rows that exist are returned and validation reports the
// missing data.
func contiguousPix(img *image.RGBA) []uint8 {
	width, height := img.Rect.Dx(), img.Rect.Dy()
	rowLength := width * 4
	if width <= 0 || height <= 0 {
		return nil
	}
	start := img.PixOffset(img.Rect.Min.X, img.Rect.Min.Y)
	if start < 0 || start > len(img.Pix) {
		return nil
	}
	if img.Stride == rowLength {
		return img.Pix[start:min(start+rowLength*height, len(img.Pix))]
	}
	pix := make([]uint8, rowLength*height)
	for y := 0; y < height; y++ {
		rowStart := start + y*img.Stride
		if rowStart < 0 || rowStart+rowLength > len(img.Pix) {
			return pix[:y*rowLength]
		}
		copy(pix[y*rowLength:], img.Pix[rowStart:rowStart+rowLength])
	}
	return pix
}

// ToImage creates an image for all layers and depth slices for the given mipmap.
//
// Array layers and depth slices are arranged vertically from top to bottom.
func (s *SurfaceRgba8) ToImage(mipmap uint32) (*image.RGBA, error) {
	// Mipmaps have different dimensions.
	// A single 2D image can only represent data from a single mip level across layers.
	var imageData []uint8
	for layer := uint32(0); layer < s.Layers; layer++ {
		for level := uint32(0); level < MipDimension(s.Depth, mipmap); level++ {
			data := s.Get(layer, level, mipmap)
			imageData = append(imageData, data...)
		}
	}
	dataLength := len(imageData)

	// Arrange depth and array layers vertically.
	// This layout allows copyless conversions to an RGBA8 surface.
	width := MipDimension(s.Width, mipmap)
	height := MipDimension(s.Height, mipmap) * MipDimension(s.Depth, mipmap) * s.Layers

	img := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))
	if len(img.Pix) != dataLength {
		return nil, &CreateImageError{Kind: CreateImageErrorInvalidSurfaceDimensions, Width: width, Height: height, DataLength: dataLength}
	}
	copy(img.Pix, imageData)
	return img, nil
}

// IntoImage creates an image for all layers and depth slices that shares the
// surface's data without copying.
//
// Fails if the surface has more than one mipmap.
// Array layers and depth slices are arranged vertically from top to bottom.
func (s *SurfaceRgba8) IntoImage() (*image.RGBA, error) {
	// Arrange depth and array layers vertically.
	// This layout allows copyless conversions to an RGBA8 surface.
	width := s.Width
	height := s.Height * s.Depth * s.Layers

	if s.Mipmaps > 1 {
		return nil, &CreateImageError{Kind: CreateImageErrorUnexpectedMipmapCount, Mipmaps: s.Mipmaps, MaxMipmaps: 1}
	}

	dataLength := len(s.Data)
	if uint64(width)*uint64(height)*4 != uint64(dataLength) {
		return nil, &CreateImageError{Kind: CreateImageErrorInvalidSurfaceDimensions, Width: width, Height: height, DataLength: dataLength}
	}
	return &image.RGBA{Pix: s.Data, Stride: int(width) * 4, Rect: image.Rect(0, 0, int(width), int(height))}, nil
}

// Image returns the image corresponding to the specified layer, depth
// level, and mipmap, or nil if the range is not fully contained.
func (s *SurfaceRgba8) Image(layer, depthLevel, mipmap uint32) *image.RGBA {
	data := s.Get(layer, depthLevel, mipmap)
	if data == nil {
		return nil
	}
	width := MipDimension(s.Width, mipmap)
	height := MipDimension(s.Height, mipmap)
	img := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))
	if len(img.Pix) != len(data) {
		return nil
	}
	copy(img.Pix, data)
	return img
}

// ToNRGBA creates an NRGBA image for all layers and depth slices for the
// given mipmap, arranged like ToImage. Surfaces store straight
// (non-premultiplied) alpha, which matches image.NRGBA; prefer it over
// image.RGBA when compositing or encoding images with alpha.
func (s *SurfaceRgba8) ToNRGBA(mipmap uint32) (*image.NRGBA, error) {
	img, err := s.ToImage(mipmap)
	if err != nil {
		return nil, err
	}
	return &image.NRGBA{Pix: img.Pix, Stride: img.Stride, Rect: img.Rect}, nil
}
