package ddsutil

// The MIT License (MIT)
//
// Copyright (c) 2018 Michael Dilger
//
// ... (see LICENSE for the full text)

import (
	"fmt"
	"io"
)

// FourCC is a "four character code" identifying a pixel format, stored
// little-endian in the DDS pixel format structure.
type FourCC uint32

// Known FourCC codes.
const (
	FourCCNone FourCC = 0

	// D3D formats
	FourCCDXT1          FourCC = 0x31545844 // "DXT1"
	FourCCDXT2          FourCC = 0x32545844 // "DXT2"
	FourCCDXT3          FourCC = 0x33545844 // "DXT3"
	FourCCDXT4          FourCC = 0x34545844 // "DXT4"
	FourCCDXT5          FourCC = 0x35545844 // "DXT5"
	FourCCR8G8_B8G8     FourCC = 0x47424752 // "RGBG"
	FourCCG8R8_G8B8     FourCC = 0x42475247 // "GRGB"
	FourCCA16B16G16R16  FourCC = 36
	FourCCQ16W16V16U16  FourCC = 110
	FourCCR16F          FourCC = 111
	FourCCG16R16F       FourCC = 112
	FourCCA16B16G16R16F FourCC = 113
	FourCCR32F          FourCC = 114
	FourCCG32R32F       FourCC = 115
	FourCCA32B32G32R32F FourCC = 116
	FourCCUYVY          FourCC = 0x59565955 // "UYVY"
	FourCCYUY2          FourCC = 0x32595559 // "YUY2"
	FourCCCXV8U8        FourCC = 117
	FourCCATI1          FourCC = 0x31495441 // "ATI1" (BC4 unorm)
	FourCCATI2          FourCC = 0x32495441 // "ATI2" (BC5 unorm)
	FourCCDX10          FourCC = 0x30315844 // "DX10"

	// DXGI formats (different names, often for same things)
	FourCCBC1_UNORM          FourCC = 0x31545844 // "DXT1"
	FourCCBC2_UNORM          FourCC = 0x33545844 // "DXT3"
	FourCCBC3_UNORM          FourCC = 0x35545844 // "DXT5"
	FourCCBC4_UNORM          FourCC = 0x55344342 // "BC4U"
	FourCCBC4_SNORM          FourCC = 0x53344342 // "BC4S"
	FourCCBC5_UNORM          FourCC = 0x32495441 // "ATI2"
	FourCCBC5_SNORM          FourCC = 0x53354342 // "BC5S"
	FourCCR8G8_B8G8_UNORM    FourCC = 0x47424752 // "RGBG"
	FourCCG8R8_G8B8_UNORM    FourCC = 0x42475247 // "GRGB"
	FourCCR16G16B16A16_UNORM FourCC = 36
	FourCCR16G16B16A16_SNORM FourCC = 110
	FourCCR16_FLOAT          FourCC = 111
	FourCCR16G16_FLOAT       FourCC = 112
	FourCCR16G16B16A16_FLOAT FourCC = 113
	FourCCR32_FLOAT          FourCC = 114
	FourCCR32G32_FLOAT       FourCC = 115
	FourCCR32G32B32A32_FLOAT FourCC = 116
)

var fourCCNames = map[FourCC]string{
	FourCCNone:          "NONE",
	FourCCDXT1:          "DXT1",
	FourCCDXT2:          "DXT2",
	FourCCDXT3:          "DXT3",
	FourCCDXT4:          "DXT4",
	FourCCDXT5:          "DXT5",
	FourCCR8G8_B8G8:     "R8G8_B8G8",
	FourCCG8R8_G8B8:     "G8R8_G8B8",
	FourCCA16B16G16R16:  "A16B16G16R16",
	FourCCQ16W16V16U16:  "Q16W16V16U16",
	FourCCR16F:          "R16F",
	FourCCG16R16F:       "G16R16F",
	FourCCA16B16G16R16F: "A16B16G16R16F",
	FourCCR32F:          "R32F",
	FourCCG32R32F:       "G32R32F",
	FourCCA32B32G32R32F: "A32B32G32R32F",
	FourCCUYVY:          "UYVY",
	FourCCYUY2:          "YUY2",
	FourCCCXV8U8:        "CXV8U8",
	FourCCATI1:          "ATI1",
	FourCCATI2:          "ATI2",
	FourCCDX10:          "DX10",
	// Note: several DXGI FourCC aliases share the same numeric value as the
	// D3D codes above (e.g. BC1_UNORM == DXT1), so only unique names are
	// listed here.
	FourCCBC4_UNORM: "BC4_UNORM",
	FourCCBC4_SNORM: "BC4_SNORM",
	FourCCBC5_SNORM: "BC5_SNORM",
}

func (f FourCC) String() string {
	if s, ok := fourCCNames[f]; ok {
		return s
	}
	return fmt.Sprintf("FourCC(0x%08x)", uint32(f))
}

// PixelFormatFlags indicate what type of data is in the surface.
type PixelFormatFlags uint32

const (
	// PixelFormatFlagsALPHA_PIXELS: texture contains alpha data.
	PixelFormatFlagsALPHA_PIXELS PixelFormatFlags = 0x1
	// PixelFormatFlagsALPHA: alpha channel only uncompressed data (used in
	// older DDS files).
	PixelFormatFlagsALPHA PixelFormatFlags = 0x2
	// PixelFormatFlagsFOURCC: texture contains compressed RGB data.
	PixelFormatFlagsFOURCC PixelFormatFlags = 0x4
	// PixelFormatFlagsRGB: texture contains uncompressed RGB data.
	PixelFormatFlagsRGB PixelFormatFlags = 0x40
	// PixelFormatFlagsYUV: YUV uncompressed data (used in older DDS files).
	PixelFormatFlagsYUV PixelFormatFlags = 0x200
	// PixelFormatFlagsLUMINANCE: single channel color uncompressed data
	// (used in older DDS files).
	PixelFormatFlagsLUMINANCE PixelFormatFlags = 0x20000

	pixelFormatFlagsAll PixelFormatFlags = PixelFormatFlagsALPHA_PIXELS |
		PixelFormatFlagsALPHA | PixelFormatFlagsFOURCC | PixelFormatFlagsRGB |
		PixelFormatFlagsYUV | PixelFormatFlagsLUMINANCE
)

// Contains reports whether all bits in o are set.
func (f PixelFormatFlags) Contains(o PixelFormatFlags) bool { return f&o == o }

// Insert sets the bits in o.
func (f *PixelFormatFlags) Insert(o PixelFormatFlags) { *f |= o }

// Bits returns the raw bit value.
func (f PixelFormatFlags) Bits() uint32 { return uint32(f) }

func (f PixelFormatFlags) String() string {
	return formatFlags(f, []flagName[PixelFormatFlags]{
		{PixelFormatFlagsALPHA_PIXELS, "ALPHA_PIXELS"},
		{PixelFormatFlagsALPHA, "ALPHA"},
		{PixelFormatFlagsFOURCC, "FOURCC"},
		{PixelFormatFlagsRGB, "RGB"},
		{PixelFormatFlagsYUV, "YUV"},
		{PixelFormatFlagsLUMINANCE, "LUMINANCE"},
	})
}

// PixelFormat describes the pixel format of the surfaces in the file.
// Optional fields are represented as pointers (nil means "not present",
// which can be significant for format detection).
type PixelFormat struct {
	// Size of this structure in bytes; set to 32.
	Size uint32

	// Values which indicate what type of data is in the surface.
	Flags PixelFormatFlags

	// Codes for specifying compressed or custom formats.
	FourCC *FourCC

	// Number of bits in an RGB (possibly including alpha) format. Valid when
	// flags includes RGB or LUMINANCE.
	RGBBitCount *uint32

	// Red (or Y) mask for reading color data. For instance, given the
	// A8R8G8B8 format, the red mask would be 0x00ff0000.
	RBitMask *uint32

	// Green (or U) mask for reading color data. For instance, given the
	// A8R8G8B8 format, the green mask would be 0x0000ff00.
	GBitMask *uint32

	// Blue (or V) mask for reading color data. For instance, given the
	// A8R8G8B8 format, the blue mask would be 0x000000ff.
	BBitMask *uint32

	// Alpha mask for reading alpha data. Valid when flags includes
	// ALPHA_PIXELS or ALPHA. For instance, given the A8R8G8B8 format, the
	// alpha mask would be 0xff000000.
	ABitMask *uint32
}

// DefaultPixelFormat returns the default (empty) pixel format, size 32.
func DefaultPixelFormat() PixelFormat {
	return PixelFormat{Size: 32} // must be 32
}

// ReadPixelFormat reads a PixelFormat structure from r.
func ReadPixelFormat(r io.Reader) (PixelFormat, error) {
	var pf PixelFormat
	size, err := readU32(r)
	if err != nil {
		return pf, err
	}
	if size != 32 {
		return pf, InvalidFieldError("Pixel format struct size")
	}
	flags, err := readU32(r)
	if err != nil {
		return pf, err
	}
	fourcc, err := readU32(r)
	if err != nil {
		return pf, err
	}
	rgbBitCount, err := readU32(r)
	if err != nil {
		return pf, err
	}
	rBitMask, err := readU32(r)
	if err != nil {
		return pf, err
	}
	gBitMask, err := readU32(r)
	if err != nil {
		return pf, err
	}
	bBitMask, err := readU32(r)
	if err != nil {
		return pf, err
	}
	aBitMask, err := readU32(r)
	if err != nil {
		return pf, err
	}
	flagsParsed := PixelFormatFlags(flags) & pixelFormatFlagsAll
	pf = PixelFormat{Size: size, Flags: flagsParsed}
	if flagsParsed.Contains(PixelFormatFlagsFOURCC) {
		fc := FourCC(fourcc)
		pf.FourCC = &fc
	}
	if flagsParsed.Contains(PixelFormatFlagsRGB) || flagsParsed.Contains(PixelFormatFlagsLUMINANCE) {
		pf.RGBBitCount = &rgbBitCount
	}
	if flagsParsed.Contains(PixelFormatFlagsRGB) {
		pf.RBitMask = &rBitMask
		pf.GBitMask = &gBitMask
		pf.BBitMask = &bBitMask
	}
	if flagsParsed.Contains(PixelFormatFlagsALPHA_PIXELS) || flagsParsed.Contains(PixelFormatFlagsALPHA) {
		pf.ABitMask = &aBitMask
	}
	return pf, nil
}

// Write writes the PixelFormat structure to w.
func (pf *PixelFormat) Write(w io.Writer) error {
	if err := writeU32(w, pf.Size); err != nil {
		return err
	}
	if err := writeU32(w, pf.Flags.Bits()); err != nil {
		return err
	}
	var fc uint32
	if pf.FourCC != nil {
		fc = uint32(*pf.FourCC)
	}
	if err := writeU32(w, fc); err != nil {
		return err
	}
	if err := writeU32(w, u32Val(pf.RGBBitCount, 0)); err != nil {
		return err
	}
	if err := writeU32(w, u32Val(pf.RBitMask, 0)); err != nil {
		return err
	}
	if err := writeU32(w, u32Val(pf.GBitMask, 0)); err != nil {
		return err
	}
	if err := writeU32(w, u32Val(pf.BBitMask, 0)); err != nil {
		return err
	}
	if err := writeU32(w, u32Val(pf.ABitMask, 0)); err != nil {
		return err
	}
	return nil
}

func (pf *PixelFormat) String() string {
	return pf.DebugString()
}

func (pf *PixelFormat) DebugString() string {
	return fmt.Sprintf(
		"    Pixel Format:\n      flags: %s\n      fourcc: %s\n      bits_per_pixel: %s\n      RGBA bitmasks: %s, %s, %s, %s\n",
		pf.Flags,
		optFourCC(pf.FourCC),
		optU32(pf.RGBBitCount),
		optU32(pf.RBitMask),
		optU32(pf.GBitMask),
		optU32(pf.BBitMask),
		optU32(pf.ABitMask),
	)
}

func optU32(p *uint32) string {
	if p == nil {
		return "None"
	}
	return fmt.Sprintf("Some(%d)", *p)
}

func optFourCC(p *FourCC) string {
	if p == nil {
		return "None"
	}
	return fmt.Sprintf("Some(%s)", *p)
}

// PixelFormatFromD3D builds a PixelFormat describing a D3DFormat.
func PixelFormatFromD3D(format D3DFormat) PixelFormat {
	pf := DefaultPixelFormat()
	if bpp, ok := format.GetBitsPerPixel(); ok {
		pf.Flags.Insert(PixelFormatFlagsRGB)
		pf.RGBBitCount = u32p(uint32(bpp))
	} else if fc, ok := format.GetFourCC(); ok {
		pf.Flags.Insert(PixelFormatFlagsFOURCC)
		pf.FourCC = &fc
	}
	if abitmask, ok := format.ABitMask(); ok {
		pf.Flags.Insert(PixelFormatFlagsALPHA_PIXELS)
		pf.ABitMask = u32p(abitmask)
	}
	pf.RBitMask = maskPtr(format.RBitMask())
	pf.GBitMask = maskPtr(format.GBitMask())
	pf.BBitMask = maskPtr(format.BBitMask())
	return pf
}

// PixelFormatFromDXGI builds a PixelFormat describing a DxgiFormat. We always
// use the DX10 extension for DXGI formats.
func PixelFormatFromDXGI(format DxgiFormat) PixelFormat {
	pf := DefaultPixelFormat()
	if bpp, ok := format.GetBitsPerPixel(); ok {
		pf.Flags.Insert(PixelFormatFlagsRGB) // means uncompressed
		pf.RGBBitCount = u32p(uint32(bpp))
	}
	fc := FourCC(FourCCDX10) // we always use extension for Dxgi
	pf.FourCC = &fc
	pf.Flags.Insert(PixelFormatFlagsFOURCC)

	// ALPHA_PIXELS is not set, use DX10 extension.
	// RBitMask, GBitMask, BBitMask and ABitMask are not set.
	return pf
}

func maskPtr(v uint32, ok bool) *uint32 {
	if !ok {
		return nil
	}
	return u32p(v)
}
