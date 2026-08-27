package ddsutil

// The MIT License (MIT)
//
// Copyright (c) 2018 Michael Dilger
//
// ... (see LICENSE for the full text)

import "fmt"

// D3DFormat is a legacy Direct3D pixel format.
type D3DFormat int

// D3D formats, in enumeration order.
const (
	D3DFormatA8B8G8R8 D3DFormat = iota
	D3DFormatG16R16
	D3DFormatA2B10G10R10
	D3DFormatA1R5G5B5
	D3DFormatR5G6B5
	D3DFormatA8
	D3DFormatA8R8G8B8
	D3DFormatX8R8G8B8
	D3DFormatX8B8G8R8
	D3DFormatA2R10G10B10
	D3DFormatR8G8B8
	D3DFormatX1R5G5B5
	D3DFormatA4R4G4B4
	D3DFormatX4R4G4B4
	D3DFormatA8R3G3B2
	D3DFormatA8L8
	D3DFormatL16
	D3DFormatL8
	D3DFormatA4L4
	D3DFormatDXT1
	D3DFormatDXT3
	D3DFormatDXT5
	D3DFormatR8G8_B8G8
	D3DFormatG8R8_G8B8
	D3DFormatA16B16G16R16
	D3DFormatQ16W16V16U16
	D3DFormatR16F
	D3DFormatG16R16F
	D3DFormatA16B16G16R16F
	D3DFormatR32F
	D3DFormatG32R32F
	D3DFormatA32B32G32R32F
	D3DFormatDXT2
	D3DFormatDXT4
	D3DFormatUYVY
	D3DFormatYUY2
	D3DFormatCXV8U8
)

var d3dFormatNames = [...]string{
	"A8B8G8R8",
	"G16R16",
	"A2B10G10R10",
	"A1R5G5B5",
	"R5G6B5",
	"A8",
	"A8R8G8B8",
	"X8R8G8B8",
	"X8B8G8R8",
	"A2R10G10B10",
	"R8G8B8",
	"X1R5G5B5",
	"A4R4G4B4",
	"X4R4G4B4",
	"A8R3G3B2",
	"A8L8",
	"L16",
	"L8",
	"A4L4",
	"DXT1",
	"DXT3",
	"DXT5",
	"R8G8_B8G8",
	"G8R8_G8B8",
	"A16B16G16R16",
	"Q16W16V16U16",
	"R16F",
	"G16R16F",
	"A16B16G16R16F",
	"R32F",
	"G32R32F",
	"A32B32G32R32F",
	"DXT2",
	"DXT4",
	"UYVY",
	"YUY2",
	"CXV8U8",
}

func (f D3DFormat) String() string {
	if f >= 0 && int(f) < len(d3dFormatNames) {
		return d3dFormatNames[f]
	}
	return fmt.Sprintf("D3DFormat(%d)", int(f))
}

// GetPitch implements DataFormat. See https://msdn.microsoft.com/en-us/library/bb943991.aspx
func (f D3DFormat) GetPitch(width uint32) (uint32, bool) {
	if f == D3DFormatR8G8_B8G8 || f == D3DFormatG8R8_G8B8 {
		return ((width + 1) >> 1) * 4, true
	}
	if bpp, ok := f.GetBitsPerPixel(); ok {
		return (width*uint32(bpp) + 7) / 8, true
	}
	if bs, ok := f.GetBlockSize(); ok {
		w := (width + 3) / 4
		if w < 1 {
			w = 1
		}
		return w * bs, true
	}
	return 0, false
}

// GetPitchHeight implements DataFormat.
func (f D3DFormat) GetPitchHeight() uint32 {
	bs, ok := f.GetBlockSize()
	return defaultPitchHeight(bs, ok)
}

// GetBitsPerPixel implements DataFormat.
func (f D3DFormat) GetBitsPerPixel() (uint8, bool) {
	switch f {
	case D3DFormatA8B8G8R8:
		return 32, true
	case D3DFormatG16R16:
		return 32, true
	case D3DFormatA2B10G10R10:
		return 32, true
	case D3DFormatA1R5G5B5:
		return 16, true
	case D3DFormatR5G6B5:
		return 16, true
	case D3DFormatA8:
		return 8, true
	case D3DFormatA8R8G8B8:
		return 32, true
	case D3DFormatX8R8G8B8:
		return 32, true
	case D3DFormatX8B8G8R8:
		return 32, true
	case D3DFormatA2R10G10B10:
		return 32, true
	case D3DFormatR8G8B8:
		return 24, true
	case D3DFormatX1R5G5B5:
		return 16, true
	case D3DFormatA4R4G4B4:
		return 16, true
	case D3DFormatX4R4G4B4:
		return 16, true
	case D3DFormatA8R3G3B2:
		return 16, true
	case D3DFormatA8L8:
		return 16, true
	case D3DFormatL16:
		return 16, true
	case D3DFormatL8:
		return 8, true
	case D3DFormatA4L4:
		return 8, true
	case D3DFormatDXT1:
		return 0, false
	case D3DFormatDXT3:
		return 0, false
	case D3DFormatDXT5:
		return 0, false
	case D3DFormatR8G8_B8G8:
		return 32, true
	case D3DFormatG8R8_G8B8:
		return 32, true
	case D3DFormatA16B16G16R16:
		return 64, true
	case D3DFormatQ16W16V16U16:
		return 64, true
	case D3DFormatR16F:
		return 16, true
	case D3DFormatG16R16F:
		return 32, true
	case D3DFormatA16B16G16R16F:
		return 64, true
	case D3DFormatR32F:
		return 32, true
	case D3DFormatG32R32F:
		return 64, true
	case D3DFormatA32B32G32R32F:
		return 128, true
	case D3DFormatDXT2:
		return 0, false
	case D3DFormatDXT4:
		return 0, false
	case D3DFormatUYVY:
		return 16, true
	case D3DFormatYUY2:
		return 16, true
	case D3DFormatCXV8U8:
		return 16, true
	default:
		return 0, false
	}
}

// GetBlockSize implements DataFormat.
func (f D3DFormat) GetBlockSize() (uint32, bool) {
	switch f {
	case D3DFormatDXT1:
		return 8, true
	case D3DFormatDXT2, D3DFormatDXT3, D3DFormatDXT4, D3DFormatDXT5:
		return 16, true
	default:
		return 0, false
	}
}

// GetFourCC implements DataFormat.
func (f D3DFormat) GetFourCC() (FourCC, bool) {
	switch f {
	case D3DFormatA8B8G8R8:
		return 0, false
	case D3DFormatG16R16:
		return 0, false
	case D3DFormatA2B10G10R10:
		return 0, false
	case D3DFormatA1R5G5B5:
		return 0, false
	case D3DFormatR5G6B5:
		return 0, false
	case D3DFormatA8:
		return 0, false
	case D3DFormatA8R8G8B8:
		return 0, false
	case D3DFormatX8R8G8B8:
		return 0, false
	case D3DFormatX8B8G8R8:
		return 0, false
	case D3DFormatA2R10G10B10:
		return 0, false
	case D3DFormatR8G8B8:
		return 0, false
	case D3DFormatX1R5G5B5:
		return 0, false
	case D3DFormatA4R4G4B4:
		return 0, false
	case D3DFormatX4R4G4B4:
		return 0, false
	case D3DFormatA8R3G3B2:
		return 0, false
	case D3DFormatA8L8:
		return 0, false
	case D3DFormatL16:
		return 0, false
	case D3DFormatL8:
		return 0, false
	case D3DFormatA4L4:
		return 0, false
	case D3DFormatDXT1:
		return FourCC(FourCCDXT1), true
	case D3DFormatDXT3:
		return FourCC(FourCCDXT3), true
	case D3DFormatDXT5:
		return FourCC(FourCCDXT5), true
	case D3DFormatR8G8_B8G8:
		return FourCC(FourCCR8G8_B8G8), true
	case D3DFormatG8R8_G8B8:
		return FourCC(FourCCG8R8_G8B8), true
	case D3DFormatA16B16G16R16:
		return FourCC(FourCCA16B16G16R16), true
	case D3DFormatQ16W16V16U16:
		return FourCC(FourCCQ16W16V16U16), true
	case D3DFormatR16F:
		return FourCC(FourCCR16F), true
	case D3DFormatG16R16F:
		return FourCC(FourCCG16R16F), true
	case D3DFormatA16B16G16R16F:
		return FourCC(FourCCA16B16G16R16F), true
	case D3DFormatR32F:
		return FourCC(FourCCR32F), true
	case D3DFormatG32R32F:
		return FourCC(FourCCG32R32F), true
	case D3DFormatA32B32G32R32F:
		return FourCC(FourCCA32B32G32R32F), true
	case D3DFormatDXT2:
		return FourCC(FourCCDXT2), true
	case D3DFormatDXT4:
		return FourCC(FourCCDXT4), true
	case D3DFormatUYVY:
		return FourCC(FourCCUYVY), true
	case D3DFormatYUY2:
		return FourCC(FourCCYUY2), true
	case D3DFormatCXV8U8:
		return FourCC(FourCCCXV8U8), true
	default:
		return 0, false
	}
}

// RequiresExtension implements DataFormat. D3D formats never require the DX10
// extension.
func (f D3DFormat) RequiresExtension() bool { return false }

// GetMinimumMipmapSizeInBytes implements DataFormat.
func (f D3DFormat) GetMinimumMipmapSizeInBytes() (uint32, bool) {
	bpp, hasBPP := f.GetBitsPerPixel()
	bs, hasBS := f.GetBlockSize()
	return defaultMinimumMipmapSizeInBytes(bpp, hasBPP, bs, hasBS)
}

// RBitMask gets the bitmask for the red channel pixels.
func (f D3DFormat) RBitMask() (uint32, bool) {
	switch f {
	case D3DFormatA8B8G8R8:
		return 0x0000_00ff, true
	case D3DFormatG16R16:
		return 0x0000_ffff, true
	case D3DFormatA2B10G10R10:
		return 0x0000_03ff, true
	case D3DFormatA1R5G5B5:
		return 0x7c00, true
	case D3DFormatR5G6B5:
		return 0xf800, true
	case D3DFormatA8:
		return 0, false
	case D3DFormatA8R8G8B8:
		return 0x00ff_0000, true
	case D3DFormatX8R8G8B8:
		return 0x00ff_0000, true
	case D3DFormatX8B8G8R8:
		return 0x0000_00ff, true
	case D3DFormatA2R10G10B10:
		return 0x3ff0_0000, true
	case D3DFormatR8G8B8:
		return 0xff_0000, true
	case D3DFormatX1R5G5B5:
		return 0x7c00, true
	case D3DFormatA4R4G4B4:
		return 0x0f00, true
	case D3DFormatX4R4G4B4:
		return 0x0f00, true
	case D3DFormatA8R3G3B2:
		return 0x00e0, true
	case D3DFormatA8L8:
		return 0x00ff, true
	case D3DFormatL16:
		return 0xffff, true
	case D3DFormatL8:
		return 0xff, true
	case D3DFormatA4L4:
		return 0x0f, true
	default:
		return 0, false
	}
}

// GBitMask gets the bitmask for the green channel pixels.
func (f D3DFormat) GBitMask() (uint32, bool) {
	switch f {
	case D3DFormatA8B8G8R8:
		return 0x0000_ff00, true
	case D3DFormatG16R16:
		return 0xffff_0000, true
	case D3DFormatA2B10G10R10:
		return 0x000f_fc00, true
	case D3DFormatA1R5G5B5:
		return 0x03e0, true
	case D3DFormatR5G6B5:
		return 0x07e0, true
	case D3DFormatA8:
		return 0, false
	case D3DFormatA8R8G8B8:
		return 0x0000_ff00, true
	case D3DFormatX8R8G8B8:
		return 0x0000_ff00, true
	case D3DFormatX8B8G8R8:
		return 0x0000_ff00, true
	case D3DFormatA2R10G10B10:
		return 0x000f_fc00, true
	case D3DFormatR8G8B8:
		return 0x00_ff00, true
	case D3DFormatX1R5G5B5:
		return 0x03e0, true
	case D3DFormatA4R4G4B4:
		return 0x00f0, true
	case D3DFormatX4R4G4B4:
		return 0x00f0, true
	case D3DFormatA8R3G3B2:
		return 0x001c, true
	case D3DFormatA8L8:
		return 0, false
	case D3DFormatL16:
		return 0, false
	case D3DFormatL8:
		return 0, false
	case D3DFormatA4L4:
		return 0, false
	default:
		return 0, false
	}
}

// BBitMask gets the bitmask for the blue channel pixels.
func (f D3DFormat) BBitMask() (uint32, bool) {
	switch f {
	case D3DFormatA8B8G8R8:
		return 0x00ff_0000, true
	case D3DFormatG16R16:
		return 0, false
	case D3DFormatA2B10G10R10:
		return 0x3ff0_0000, true
	case D3DFormatA1R5G5B5:
		return 0x001f, true
	case D3DFormatR5G6B5:
		return 0x001f, true
	case D3DFormatA8:
		return 0, false
	case D3DFormatA8R8G8B8:
		return 0x0000_00ff, true
	case D3DFormatX8R8G8B8:
		return 0x0000_00ff, true
	case D3DFormatX8B8G8R8:
		return 0x00ff_0000, true
	case D3DFormatA2R10G10B10:
		return 0x0000_03ff, true
	case D3DFormatR8G8B8:
		return 0x00_00ff, true
	case D3DFormatX1R5G5B5:
		return 0x001f, true
	case D3DFormatA4R4G4B4:
		return 0x000f, true
	case D3DFormatX4R4G4B4:
		return 0x000f, true
	case D3DFormatA8R3G3B2:
		return 0x0003, true
	case D3DFormatA8L8:
		return 0, false
	case D3DFormatL16:
		return 0, false
	case D3DFormatL8:
		return 0, false
	case D3DFormatA4L4:
		return 0, false
	default:
		return 0, false
	}
}

// ABitMask gets the bitmask for the alpha channel pixels.
func (f D3DFormat) ABitMask() (uint32, bool) {
	switch f {
	case D3DFormatA8B8G8R8:
		return 0xff00_0000, true
	case D3DFormatG16R16:
		return 0, false
	case D3DFormatA2B10G10R10:
		return 0xc000_0000, true
	case D3DFormatA1R5G5B5:
		return 0x8000, true
	case D3DFormatR5G6B5:
		return 0, false
	case D3DFormatA8:
		return 0xff, true
	case D3DFormatA8R8G8B8:
		return 0xff00_0000, true
	case D3DFormatX8R8G8B8:
		return 0, false
	case D3DFormatX8B8G8R8:
		return 0, false
	case D3DFormatA2R10G10B10:
		return 0xc000_0000, true
	case D3DFormatR8G8B8:
		return 0, false
	case D3DFormatX1R5G5B5:
		return 0, false
	case D3DFormatA4R4G4B4:
		return 0xf000, true
	case D3DFormatX4R4G4B4:
		return 0, false
	case D3DFormatA8R3G3B2:
		return 0xff00, true
	case D3DFormatA8L8:
		return 0xff00, true
	case D3DFormatL16:
		return 0, false
	case D3DFormatL8:
		return 0, false
	case D3DFormatA4L4:
		return 0xf0, true
	default:
		return 0, false
	}
}

// d3dPixelFormatPattern is one row of the pixel-format-to-D3DFormat matching
// table. Nil mask pointers mean the field must be absent; non-nil pointers
// must match exactly.
type d3dPixelFormatPattern struct {
	lum         bool
	rgb         bool
	alpha       bool
	bpp         uint32
	bppMayBeNil bool // allows RGBBitCount to be absent (the A8 row)
	r           *uint32
	g           *uint32
	b           *uint32
	a           *uint32
	format      D3DFormat
}

func maskEq(pattern, v *uint32) bool {
	if pattern == nil {
		return v == nil
	}
	return v != nil && *v == *pattern
}

func (p *d3dPixelFormatPattern) matches(lum, rgb, alpha bool, bpp, r, g, b, a *uint32) bool {
	if p.lum != lum || p.rgb != rgb || p.alpha != alpha {
		return false
	}
	bppOK := (bpp != nil && *bpp == p.bpp) || (p.bppMayBeNil && bpp == nil)
	if !bppOK {
		return false
	}
	return maskEq(p.r, r) && maskEq(p.g, g) && maskEq(p.b, b) && maskEq(p.a, a)
}

// d3dPixelFormatPatterns is the pixel-format-to-D3DFormat match table, in order.
var d3dPixelFormatPatterns = []d3dPixelFormatPattern{
	// lum   rgb   alpha bpp  anybpp r                g                b                a
	{false, true, true, 32, false, u32p(0xff), u32p(0xff00), u32p(0xff0000), u32p(0xff000000), D3DFormatA8B8G8R8},
	{false, true, false, 32, false, u32p(0xffff), u32p(0xffff0000), nil, nil, D3DFormatG16R16},
	{false, true, true, 32, false, u32p(0x3ff), u32p(0xffc00), u32p(0x3ff00000), nil, D3DFormatA2B10G10R10},
	{false, true, true, 16, false, u32p(0x7c00), u32p(0x3e0), u32p(0x1f), u32p(0x8000), D3DFormatA1R5G5B5},
	{false, true, false, 16, false, u32p(0xf800), u32p(0x7e0), u32p(0x1f), nil, D3DFormatR5G6B5},
	{false, false, true, 8, true, nil, nil, nil, u32p(0xff), D3DFormatA8}, // bpp MayBeNil
	{false, true, true, 32, false, u32p(0xff0000), u32p(0xff00), u32p(0xff), u32p(0xff000000), D3DFormatA8R8G8B8},
	{false, true, false, 32, false, u32p(0xff0000), u32p(0xff00), u32p(0xff), nil, D3DFormatX8R8G8B8},
	{false, true, false, 32, false, u32p(0xff), u32p(0xff00), u32p(0xff0000), nil, D3DFormatX8B8G8R8},
	{false, true, true, 32, false, u32p(0x3ff00000), u32p(0xffc00), u32p(0x3ff), u32p(0xc0000000), D3DFormatA2R10G10B10},
	{false, true, false, 24, false, u32p(0xff0000), u32p(0xff00), u32p(0xff), nil, D3DFormatR8G8B8},
	{false, true, false, 16, false, u32p(0x7c00), u32p(0x3e0), u32p(0x1f), nil, D3DFormatX1R5G5B5},
	{false, true, true, 16, false, u32p(0xf00), u32p(0xf0), u32p(0xf), u32p(0xf000), D3DFormatA4R4G4B4},
	{false, true, false, 16, false, u32p(0xf00), u32p(0xf0), u32p(0xf), nil, D3DFormatX4R4G4B4},
	{false, true, true, 16, false, u32p(0xe0), u32p(0x1c), u32p(0x3), u32p(0xff00), D3DFormatA8R3G3B2},
	{true, false, true, 16, false, u32p(0xff), nil, nil, u32p(0xff00), D3DFormatA8L8},
	{true, false, false, 16, false, u32p(0xffff), nil, nil, nil, D3DFormatL16},
	{true, false, false, 8, false, u32p(0xff), nil, nil, nil, D3DFormatL8},
	{true, false, true, 8, false, u32p(0xf), nil, nil, u32p(0xf0), D3DFormatA4L4},
}

// D3DFormatTryFromPixelFormat attempts to use PixelFormat data (e.g. from
// dds.Header.SPF) to determine the D3DFormat.
func D3DFormatTryFromPixelFormat(pixelFormat *PixelFormat) (D3DFormat, bool) {
	if pixelFormat.FourCC != nil {
		switch *pixelFormat.FourCC {
		case FourCCDXT1:
			return D3DFormatDXT1, true
		case FourCCDXT2:
			return D3DFormatDXT2, true
		case FourCCDXT3:
			return D3DFormatDXT3, true
		case FourCCDXT4:
			return D3DFormatDXT4, true
		case FourCCDXT5:
			return D3DFormatDXT5, true
		case FourCCR8G8_B8G8:
			return D3DFormatR8G8_B8G8, true
		case FourCCG8R8_G8B8:
			return D3DFormatG8R8_G8B8, true
		case FourCCA16B16G16R16:
			return D3DFormatA16B16G16R16, true
		case FourCCQ16W16V16U16:
			return D3DFormatQ16W16V16U16, true
		case FourCCR16F:
			return D3DFormatR16F, true
		case FourCCG16R16F:
			return D3DFormatG16R16F, true
		case FourCCA16B16G16R16F:
			return D3DFormatA16B16G16R16F, true
		case FourCCR32F:
			return D3DFormatR32F, true
		case FourCCG32R32F:
			return D3DFormatG32R32F, true
		case FourCCA32B32G32R32F:
			return D3DFormatA32B32G32R32F, true
		case FourCCUYVY:
			return D3DFormatUYVY, true
		case FourCCYUY2:
			return D3DFormatYUY2, true
		case FourCCCXV8U8:
			return D3DFormatCXV8U8, true
		case FourCCDX10:
			return 0, false // should use the header10 extension instead
		default:
			return 0, false
		}
	}
	rgb := pixelFormat.Flags.Contains(PixelFormatFlagsRGB)
	alpha := pixelFormat.Flags.Contains(PixelFormatFlagsALPHA) ||
		pixelFormat.Flags.Contains(PixelFormatFlagsALPHA_PIXELS)
	lum := pixelFormat.Flags.Contains(PixelFormatFlagsLUMINANCE)
	for i := range d3dPixelFormatPatterns {
		if d3dPixelFormatPatterns[i].matches(lum, rgb, alpha,
			pixelFormat.RGBBitCount, pixelFormat.RBitMask, pixelFormat.GBitMask,
			pixelFormat.BBitMask, pixelFormat.ABitMask) {
			return d3dPixelFormatPatterns[i].format, true
		}
	}
	return 0, false
}
