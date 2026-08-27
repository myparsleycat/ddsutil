package ddsutil

// The MIT License (MIT)
//
// Copyright (c) 2018 Michael Dilger
//
// ... (see LICENSE for the full text)

// DataFormat is implemented by both D3DFormat and DxgiFormat. Methods that
// may have no meaningful value return (T, bool).
type DataFormat interface {
	// GetPitch gets the number of bytes required to store one row of data.
	GetPitch(width uint32) (uint32, bool)

	// GetPitchHeight gets the height of each row of data. Normally it is 1,
	// but for block compressed textures, each row is 4 pixels high.
	GetPitchHeight() uint32

	// GetBitsPerPixel gets the number of bits required to store a single
	// pixel. It is only defined for uncompressed formats.
	GetBitsPerPixel() (uint8, bool)

	// GetBlockSize gets a block compression format's block size, and is only
	// defined for compressed formats.
	GetBlockSize() (uint32, bool)

	// GetFourCC gets the fourcc code for this format, if known.
	GetFourCC() (FourCC, bool)

	// RequiresExtension returns true if the DX10 extension is required to
	// use this format.
	RequiresExtension() bool

	// GetMinimumMipmapSizeInBytes gets the minimum mipmap size in bytes.
	// Even if they go all the way down to 1x1, there is a minimum number of
	// bytes based on bits per pixel or blocksize.
	GetMinimumMipmapSizeInBytes() (uint32, bool)
}

// defaultPitchHeight is the shared default implementation of GetPitchHeight.
func defaultPitchHeight(blockSize uint32, hasBlockSize bool) uint32 {
	if hasBlockSize {
		return 4
	}
	return 1
}

// defaultMinimumMipmapSizeInBytes is the shared default implementation of
// GetMinimumMipmapSizeInBytes.
func defaultMinimumMipmapSizeInBytes(bpp uint8, hasBPP bool, blockSize uint32, hasBlockSize bool) (uint32, bool) {
	if hasBPP {
		return (uint32(bpp) + 7) / 8, true
	}
	if hasBlockSize {
		return blockSize, true
	}
	return 0, false
}
