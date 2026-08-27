package ddsutil

// The MIT License (MIT)
//
// Copyright (c) 2018 Michael Dilger
//
// ... (see LICENSE for the full text)

import (
	"fmt"
	"io"
	"strings"
)

// D3D10ResourceDimension identifies the resource type in the DX10 header
// extension.
type D3D10ResourceDimension uint32

const (
	D3D10ResourceDimensionUnknown   D3D10ResourceDimension = 0
	D3D10ResourceDimensionBuffer    D3D10ResourceDimension = 1
	D3D10ResourceDimensionTexture1D D3D10ResourceDimension = 2
	D3D10ResourceDimensionTexture2D D3D10ResourceDimension = 3
	D3D10ResourceDimensionTexture3D D3D10ResourceDimension = 4
)

func (d D3D10ResourceDimension) String() string {
	switch d {
	case D3D10ResourceDimensionUnknown:
		return "Unknown"
	case D3D10ResourceDimensionBuffer:
		return "Buffer"
	case D3D10ResourceDimensionTexture1D:
		return "Texture1D"
	case D3D10ResourceDimensionTexture2D:
		return "Texture2D"
	case D3D10ResourceDimensionTexture3D:
		return "Texture3D"
	default:
		return fmt.Sprintf("D3D10ResourceDimension(%d)", uint32(d))
	}
}

// D3D10ResourceDimensionFromU32 converts a raw value, returning false if
// unknown.
func D3D10ResourceDimensionFromU32(v uint32) (D3D10ResourceDimension, bool) {
	switch D3D10ResourceDimension(v) {
	case D3D10ResourceDimensionUnknown,
		D3D10ResourceDimensionBuffer,
		D3D10ResourceDimensionTexture1D,
		D3D10ResourceDimensionTexture2D,
		D3D10ResourceDimensionTexture3D:
		return D3D10ResourceDimension(v), true
	default:
		return 0, false
	}
}

// MiscFlag holds the DX10 header misc flags.
type MiscFlag uint32

// MiscFlagTEXTURECUBE: 2D Texture is a cube-map texture.
const MiscFlagTEXTURECUBE MiscFlag = 0x4

// Contains reports whether all bits in o are set.
func (f MiscFlag) Contains(o MiscFlag) bool { return f&o == o }

// Insert sets the bits in o.
func (f *MiscFlag) Insert(o MiscFlag) { *f |= o }

// Bits returns the raw bit value.
func (f MiscFlag) Bits() uint32 { return uint32(f) }

func (f MiscFlag) String() string {
	return formatFlags(f, []flagName[MiscFlag]{
		{MiscFlagTEXTURECUBE, "TEXTURECUBE"},
	})
}

// AlphaMode is the alpha mode stored in misc_flags2 of the DX10 header
// extension.
type AlphaMode uint32

const (
	AlphaModeUnknown       AlphaMode = 0x0
	AlphaModeStraight      AlphaMode = 0x1
	AlphaModePreMultiplied AlphaMode = 0x2
	AlphaModeOpaque        AlphaMode = 0x3
	AlphaModeCustom        AlphaMode = 0x4
)

func (a AlphaMode) String() string {
	switch a {
	case AlphaModeUnknown:
		return "Unknown"
	case AlphaModeStraight:
		return "Straight"
	case AlphaModePreMultiplied:
		return "PreMultiplied"
	case AlphaModeOpaque:
		return "Opaque"
	case AlphaModeCustom:
		return "Custom"
	default:
		return fmt.Sprintf("AlphaMode(%d)", uint32(a))
	}
}

// AlphaModeFromU32 converts a raw value, returning false if unknown.
func AlphaModeFromU32(v uint32) (AlphaMode, bool) {
	switch AlphaMode(v) {
	case AlphaModeUnknown,
		AlphaModeStraight,
		AlphaModePreMultiplied,
		AlphaModeOpaque,
		AlphaModeCustom:
		return AlphaMode(v), true
	default:
		return 0, false
	}
}

// Header10 is the optional DX10 extension header that follows the main
// header when the pixel format FourCC is "DX10".
type Header10 struct {
	DxgiFormat        DxgiFormat
	ResourceDimension D3D10ResourceDimension
	MiscFlag          MiscFlag
	ArraySize         uint32
	// AlphaMode is called misc_flags2 in the official documentation.
	AlphaMode AlphaMode
}

// NewHeader10 creates a new DX10 extension header.
func NewHeader10(
	format DxgiFormat,
	isCubemap bool,
	resourceDimension D3D10ResourceDimension,
	arraySize uint32,
	alphaMode AlphaMode,
) Header10 {
	var flags MiscFlag
	if isCubemap {
		flags.Insert(MiscFlagTEXTURECUBE)
	}
	return Header10{
		DxgiFormat:        format,
		ResourceDimension: resourceDimension,
		MiscFlag:          flags,
		ArraySize:         arraySize,
		AlphaMode:         alphaMode,
	}
}

// ReadHeader10 reads a Header10 structure from r.
func ReadHeader10(r io.Reader) (*Header10, error) {
	dxgiFormatRaw, err := readU32(r)
	if err != nil {
		return nil, err
	}
	resourceDimensionRaw, err := readU32(r)
	if err != nil {
		return nil, err
	}
	miscFlagRaw, err := readU32(r)
	if err != nil {
		return nil, err
	}
	arraySize, err := readU32(r)
	if err != nil {
		return nil, err
	}
	alphaModeRaw, err := readU32(r)
	if err != nil {
		return nil, err
	}

	dxgiFormat, ok := DxgiFormatFromU32(dxgiFormatRaw)
	if !ok {
		return nil, InvalidFieldError("dxgi_format")
	}
	resourceDimension, ok := D3D10ResourceDimensionFromU32(resourceDimensionRaw)
	if !ok {
		return nil, InvalidFieldError("resource_dimension")
	}
	alphaMode, ok := AlphaModeFromU32(alphaModeRaw)
	if !ok {
		return nil, InvalidFieldError("alpha mode (misc_flags2)")
	}

	return &Header10{
		DxgiFormat:        dxgiFormat,
		ResourceDimension: resourceDimension,
		MiscFlag:          MiscFlag(miscFlagRaw) & miscFlagAll,
		ArraySize:         arraySize,
		AlphaMode:         alphaMode,
	}, nil
}

// Write writes the Header10 structure to w.
func (h *Header10) Write(w io.Writer) error {
	if err := writeU32(w, uint32(h.DxgiFormat)); err != nil {
		return err
	}
	if err := writeU32(w, uint32(h.ResourceDimension)); err != nil {
		return err
	}
	if err := writeU32(w, h.MiscFlag.Bits()); err != nil {
		return err
	}
	if err := writeU32(w, h.ArraySize); err != nil {
		return err
	}
	if err := writeU32(w, uint32(h.AlphaMode)); err != nil {
		return err
	}
	return nil
}

func (h *Header10) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "  Header10:\n")
	fmt.Fprintf(&b, "    dxgi_format: %s\n", h.DxgiFormat)
	fmt.Fprintf(&b, "    resource_dimension: %s\n", h.ResourceDimension)
	fmt.Fprintf(&b, "    misc_flag: %s\n", h.MiscFlag)
	fmt.Fprintf(&b, "    array_size: %d\n", h.ArraySize)
	fmt.Fprintf(&b, "    alpha_mode: %s", h.AlphaMode)
	return b.String()
}

const miscFlagAll MiscFlag = MiscFlagTEXTURECUBE
