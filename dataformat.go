package ddsutil

// The MIT License (MIT)
//
// Copyright (c) 2018 Michael Dilger
//
// ... (see LICENSE for the full text)

// DataFormat is implemented by both D3DFormat and DxgiFormat. Methods that
// may have no meaningful value return (T, bool).
type DataFormat interface {
	// Pitch gets the number of bytes required to store one row of data.
	Pitch(width uint32) (uint32, bool)

	// PitchHeight gets the height of each row of data. Normally it is 1,
	// but for block compressed textures, each row is 4 pixels high.
	PitchHeight() uint32

	// BitsPerPixel gets the number of bits required to store a single
	// pixel. It is only defined for uncompressed formats.
	BitsPerPixel() (uint8, bool)

	// BlockSize gets a block compression format's block size, and is only
	// defined for compressed formats.
	BlockSize() (uint32, bool)

	// FourCC gets the fourcc code for this format, if known.
	FourCC() (FourCC, bool)

	// RequiresExtension returns true if the DX10 extension is required to
	// use this format.
	RequiresExtension() bool

	// MinimumMipmapSizeInBytes gets the minimum mipmap size in bytes.
	// Even if they go all the way down to 1x1, there is a minimum number of
	// bytes based on bits per pixel or blocksize.
	MinimumMipmapSizeInBytes() (uint32, bool)
}

// defaultPitchHeight is the shared default implementation of PitchHeight.
func defaultPitchHeight(blockSize uint32, hasBlockSize bool) uint32 {
	if hasBlockSize {
		return 4
	}
	return 1
}

// defaultMinimumMipmapSizeInBytes is the shared default implementation of
// MinimumMipmapSizeInBytes.
func defaultMinimumMipmapSizeInBytes(bpp uint8, hasBPP bool, blockSize uint32, hasBlockSize bool) (uint32, bool) {
	if hasBPP {
		return (uint32(bpp) + 7) / 8, true
	}
	if hasBlockSize {
		return blockSize, true
	}
	return 0, false
}

// defaultPitch is the shared implementation of Pitch. It returns false if
// the format has no pitch or the pitch does not fit in 32 bits.
func defaultPitch(width uint32, packedPair bool, bpp uint8, hasBPP bool, blockSize uint32, hasBlockSize bool) (uint32, bool) {
	var pitch uint64
	switch {
	case packedPair:
		// Two pixels share 4 bytes (R8G8_B8G8 and G8R8_G8B8).
		pitch = ((uint64(width) + 1) >> 1) * 4
	case hasBPP:
		pitch = (uint64(width)*uint64(bpp) + 7) / 8
	case hasBlockSize:
		blocks := (uint64(width) + 3) / 4
		if blocks < 1 {
			blocks = 1
		}
		pitch = blocks * uint64(blockSize)
	default:
		return 0, false
	}
	if pitch > uint64(^uint32(0)) {
		return 0, false
	}
	return uint32(pitch), true
}
