package ddsutil

import (
	"errors"
	"image"
	"image/color"
)

// ImageFromDds decodes the given mip level from the DDS to an RGBA image.
// Array layers are arranged vertically from top to bottom.
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

// SurfaceRgba8FromImage creates a 2D view over the data in img without any copies.
func SurfaceRgba8FromImage(img *image.RGBA) *SurfaceRgba8 {
	return &SurfaceRgba8{
		Width:   uint32(img.Rect.Dx()),
		Height:  uint32(img.Rect.Dy()),
		Depth:   1,
		Layers:  1,
		Mipmaps: 1,
		Data:    img.Pix,
	}
}

// SurfaceRgba8FromImageLayers creates a 2D view with layers over the data in img.
//
// Array layers should be stacked vertically in img with an overall height height*layers.
func SurfaceRgba8FromImageLayers(img *image.RGBA, layers uint32) *SurfaceRgba8 {
	return &SurfaceRgba8{
		Width:   uint32(img.Rect.Dx()),
		Height:  uint32(img.Rect.Dy()) / layers,
		Depth:   1,
		Layers:  layers,
		Mipmaps: 1,
		Data:    img.Pix,
	}
}

// SurfaceRgba8FromImageDepth creates a 3D view over the data in img.
//
// Depth slices should be stacked vertically in img with an overall height height*depth.
func SurfaceRgba8FromImageDepth(img *image.RGBA, depth uint32) *SurfaceRgba8 {
	return &SurfaceRgba8{
		Width:   uint32(img.Rect.Dx()),
		Height:  uint32(img.Rect.Dy()) / depth,
		Depth:   depth,
		Layers:  1,
		Mipmaps: 1,
		Data:    img.Pix,
	}
}

// ToImage creates an image for all layers and depth slices for the given mipmap.
//
// Array layers and depth slices are arranged vertically from top to bottom.
func (s *SurfaceRgba8) ToImage(mipmap uint32) (*image.RGBA, error) {
	// Mipmaps have different dimensions.
	// A single 2D image can only represent data from a single mip level across layers.
	var imageData []uint8
	for layer := uint32(0); layer < s.Layers; layer++ {
		for level := uint32(0); level < s.Depth; level++ {
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

// IntoImage creates an image for all layers and depth slices without copying.
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
	img := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))
	if len(img.Pix) != dataLength {
		return nil, &CreateImageError{Kind: CreateImageErrorInvalidSurfaceDimensions, Width: width, Height: height, DataLength: dataLength}
	}
	copy(img.Pix, s.Data)
	return img, nil
}

// GetImage returns the image corresponding to the specified layer, depth
// level, and mipmap, or nil if the range is not fully contained.
func (s *SurfaceRgba8) GetImage(layer, depthLevel, mipmap uint32) *image.RGBA {
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

// ToNRGBA converts the surface to an NRGBA image (values are stored
// straight, not alpha-premultiplied).
func (s *SurfaceRgba8) ToNRGBA(mipmap uint32) (*image.NRGBA, error) {
	img, err := s.ToImage(mipmap)
	if err != nil {
		return nil, err
	}
	nrgba := image.NewNRGBA(img.Rect)
	for i := 0; i < len(img.Pix); i += 4 {
		r, g, b, a := img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3]
		// Go's image.RGBA interprets the color as alpha-premultiplied when
		// converted via color.RGBA, so convert to straight alpha for NRGBA.
		_ = color.RGBA{r, g, b, a}
		nrgba.Pix[i] = r
		nrgba.Pix[i+1] = g
		nrgba.Pix[i+2] = b
		nrgba.Pix[i+3] = a
	}
	return nrgba, nil
}
