package ddsutil

// Surface is a surface with an image format known at runtime.
type Surface struct {
	// Width is the width of the surface in pixels.
	Width uint32
	// Height is the height of the surface in pixels.
	Height uint32
	// Depth is the depth of the surface in pixels. This should be 1 for 2D surfaces.
	Depth uint32
	// Layers is the number of array layers in the surface.
	// This should be 1 for most surfaces and 6 for cube maps.
	Layers uint32
	// Mipmaps is the number of mipmaps in the surface.
	// This should be 1 if the surface has only the base mip level.
	// All array layers are assumed to have the same number of mipmaps.
	Mipmaps uint32
	// ImageFormat is the format of the bytes in Data.
	ImageFormat ImageFormat
	// Data is the combined image data ordered by layer and then mipmap
	// without additional padding.
	// A surface with L layers and M mipmaps has the following layout:
	// Layer 0 Mip 0, Layer 0 Mip 1, ..., Layer L-1 Mip M-1
	Data []byte
}

// SurfaceRgba8 is an uncompressed Rgba8Unorm surface with 4 bytes per pixel.
type SurfaceRgba8 struct {
	Width   uint32
	Height  uint32
	Depth   uint32
	Layers  uint32
	Mipmaps uint32
	Data    []byte
}

// SurfaceRgba32Float is an uncompressed Rgba32Float surface with 16 bytes per pixel.
type SurfaceRgba32Float struct {
	Width   uint32
	Height  uint32
	Depth   uint32
	Layers  uint32
	Mipmaps uint32
	Data    []float32
}

// Get returns the range of image data corresponding to the specified
// layer, depthLevel, and mipmap.
//
// The dimensions of the returned data should be calculated using MipDimension.
// Returns nil if the expected range is not fully contained within the buffer.
func (s *Surface) Get(layer, depthLevel, mipmap uint32) []byte {
	return getMipmap(s.Data, [3]uint32{s.Width, s.Height, s.Depth}, s.Mipmaps, s.ImageFormat, layer, depthLevel, mipmap)
}

// Validate checks the surface dimensions and data length.
func (s *Surface) Validate() error {
	if s.Width == 0 || s.Height == 0 || s.Depth == 0 {
		return &SurfaceError{Kind: SurfaceErrorZeroSizedSurface, Width: s.Width, Height: s.Height, Depth: s.Depth}
	}

	maxMipmaps := maxMipmapCount(max3(s.Width, s.Height, s.Depth))
	if s.Mipmaps > maxMipmaps {
		return &SurfaceError{Kind: SurfaceErrorUnexpectedMipmapCount, Mipmaps: s.Mipmaps, MaxTotalMipmaps: maxMipmaps}
	}

	blockWidth, blockHeight, blockDepth := s.ImageFormat.blockDimensions()
	blockSizeInBytes := s.ImageFormat.BlockSizeInBytes()
	baseLayerSize, ok := mipSize(int(s.Width), int(s.Height), int(s.Depth), int(blockWidth), int(blockHeight), int(blockDepth), blockSizeInBytes)
	if !ok {
		return &SurfaceError{Kind: SurfaceErrorPixelCountWouldOverflow, Width: s.Width, Height: s.Height, Depth: s.Depth}
	}

	if baseLayerSize > len(s.Data) {
		return &SurfaceError{Kind: SurfaceErrorNotEnoughData, Expected: baseLayerSize, Actual: len(s.Data)}
	}

	return nil
}

func max3(a, b, c uint32) uint32 {
	m := a
	if b > m {
		m = b
	}
	if c > m {
		m = c
	}
	return m
}

// Get returns the range of 2D image data corresponding to the specified
// layer, depthLevel, and mipmap.
func (s *SurfaceRgba8) Get(layer, depthLevel, mipmap uint32) []byte {
	return getMipmap(s.Data, [3]uint32{s.Width, s.Height, s.Depth}, s.Mipmaps, Rgba8Unorm, layer, depthLevel, mipmap)
}

// Validate checks the surface dimensions and data length.
func (s *SurfaceRgba8) Validate() error {
	surf := Surface{
		Width:       s.Width,
		Height:      s.Height,
		Depth:       s.Depth,
		Layers:      s.Layers,
		Mipmaps:     s.Mipmaps,
		ImageFormat: Rgba8Unorm,
		Data:        s.Data,
	}
	return surf.Validate()
}

// Get returns the range of 2D image data corresponding to the specified
// layer, depthLevel, and mipmap.
func (s *SurfaceRgba32Float) Get(layer, depthLevel, mipmap uint32) []float32 {
	blockSizeInBytes := Rgba32Float.BlockSizeInBytes()
	bw, bh, bd := Rgba32Float.blockDimensions()
	blockDimensions := [3]uint32{bw, bh, bd}

	offsetInBytes, ok := calculateOffset(layer, depthLevel, mipmap, [3]uint32{s.Width, s.Height, s.Depth}, blockDimensions, blockSizeInBytes, s.Mipmaps)
	if !ok {
		return nil
	}

	// The returned slice is always 2D.
	mipWidth := MipDimension(s.Width, mipmap)
	mipHeight := MipDimension(s.Height, mipmap)

	sizeInBytes, ok := mipSize(int(mipWidth), int(mipHeight), 1, int(blockDimensions[0]), int(blockDimensions[1]), int(blockDimensions[2]), blockSizeInBytes)
	if !ok {
		return nil
	}

	start := offsetInBytes / 4
	count := sizeInBytes / 4
	if start+count > len(s.Data) {
		return nil
	}
	return s.Data[start : start+count]
}

// Validate checks the surface dimensions and data length.
func (s *SurfaceRgba32Float) Validate() error {
	if s.Width == 0 || s.Height == 0 || s.Depth == 0 {
		return &SurfaceError{Kind: SurfaceErrorZeroSizedSurface, Width: s.Width, Height: s.Height, Depth: s.Depth}
	}

	maxMipmaps := maxMipmapCount(max3(s.Width, s.Height, s.Depth))
	if s.Mipmaps > maxMipmaps {
		return &SurfaceError{Kind: SurfaceErrorUnexpectedMipmapCount, Mipmaps: s.Mipmaps, MaxTotalMipmaps: maxMipmaps}
	}

	blockWidth, blockHeight, blockDepth := Rgba32Float.blockDimensions()
	blockSizeInBytes := Rgba32Float.BlockSizeInBytes()
	baseLayerSize, ok := mipSize(int(s.Width), int(s.Height), int(s.Depth), int(blockWidth), int(blockHeight), int(blockDepth), blockSizeInBytes)
	if !ok {
		return &SurfaceError{Kind: SurfaceErrorPixelCountWouldOverflow, Width: s.Width, Height: s.Height, Depth: s.Depth}
	}

	if uint64(len(s.Data))*4 < uint64(baseLayerSize) {
		return &SurfaceError{Kind: SurfaceErrorNotEnoughData, Expected: int(baseLayerSize), Actual: len(s.Data) * 4}
	}

	return nil
}

// getMipmap returns the 2D image data for the specified layer, depth level,
// and mipmap, or nil when the range is out of bounds.
func getMipmap(data []byte, dimensions [3]uint32, mipmaps uint32, format ImageFormat, layer, depthLevel, mipmap uint32) []byte {
	blockSizeInBytes := format.BlockSizeInBytes()
	bw, bh, bd := format.blockDimensions()
	blockDimensions := [3]uint32{bw, bh, bd}

	offsetInBytes, ok := calculateOffset(layer, depthLevel, mipmap, dimensions, blockDimensions, blockSizeInBytes, mipmaps)
	if !ok {
		return nil
	}

	// The returned slice is always 2D.
	mipWidth := MipDimension(dimensions[0], mipmap)
	mipHeight := MipDimension(dimensions[1], mipmap)

	sizeInBytes, ok := mipSize(int(mipWidth), int(mipHeight), 1, int(blockDimensions[0]), int(blockDimensions[1]), int(blockDimensions[2]), blockSizeInBytes)
	if !ok {
		return nil
	}

	if offsetInBytes+sizeInBytes > len(data) {
		return nil
	}
	return data[offsetInBytes : offsetInBytes+sizeInBytes]
}
