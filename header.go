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

// HeaderFlags indicate which header members contain valid data.
type HeaderFlags uint32

const (
	// HeaderFlagsCAPS is required in every DDS file.
	HeaderFlagsCAPS HeaderFlags = 0x1
	// HeaderFlagsHEIGHT is required in every DDS file.
	HeaderFlagsHEIGHT HeaderFlags = 0x2
	// HeaderFlagsWIDTH is required in every DDS file.
	HeaderFlagsWIDTH HeaderFlags = 0x4
	// HeaderFlagsPITCH is required when pitch is provided for an
	// uncompressed texture.
	HeaderFlagsPITCH HeaderFlags = 0x8
	// HeaderFlagsPIXELFORMAT is required in every DDS file.
	HeaderFlagsPIXELFORMAT HeaderFlags = 0x1000
	// HeaderFlagsMIPMAPCOUNT is required in a mipmapped texture.
	HeaderFlagsMIPMAPCOUNT HeaderFlags = 0x20000
	// HeaderFlagsLINEARSIZE is required when pitch is provided for a
	// compressed texture.
	HeaderFlagsLINEARSIZE HeaderFlags = 0x80000
	// HeaderFlagsDEPTH is required in a depth texture.
	HeaderFlagsDEPTH HeaderFlags = 0x800000

	headerFlagsAll HeaderFlags = HeaderFlagsCAPS | HeaderFlagsHEIGHT |
		HeaderFlagsWIDTH | HeaderFlagsPITCH | HeaderFlagsPIXELFORMAT |
		HeaderFlagsMIPMAPCOUNT | HeaderFlagsLINEARSIZE | HeaderFlagsDEPTH
)

// Contains reports whether all bits in o are set.
func (f HeaderFlags) Contains(o HeaderFlags) bool { return f&o == o }

// Insert sets the bits in o.
func (f *HeaderFlags) Insert(o HeaderFlags) { *f |= o }

// Bits returns the raw bit value.
func (f HeaderFlags) Bits() uint32 { return uint32(f) }

// FromBits truncates unknown bits (bitflags' from_bits_truncate).
func (f HeaderFlags) FromBits(bits uint32) HeaderFlags { return HeaderFlags(bits) & headerFlagsAll }

func (f HeaderFlags) String() string {
	return formatFlags(f, []flagName[HeaderFlags]{
		{HeaderFlagsCAPS, "CAPS"},
		{HeaderFlagsHEIGHT, "HEIGHT"},
		{HeaderFlagsWIDTH, "WIDTH"},
		{HeaderFlagsPITCH, "PITCH"},
		{HeaderFlagsPIXELFORMAT, "PIXELFORMAT"},
		{HeaderFlagsMIPMAPCOUNT, "MIPMAPCOUNT"},
		{HeaderFlagsLINEARSIZE, "LINEARSIZE"},
		{HeaderFlagsDEPTH, "DEPTH"},
	})
}

// Caps specifies the complexity of the surfaces stored.
type Caps uint32

const (
	// CapsCOMPLEX must be used on any file that contains more than one
	// surface (a mipmap, a cubic environment, or a mipmapped volume texture).
	CapsCOMPLEX Caps = 0x8
	// CapsMIPMAP should be used for a mipmap.
	CapsMIPMAP Caps = 0x400000
	// CapsTEXTURE is required.
	CapsTEXTURE Caps = 0x1000

	capsAll Caps = CapsCOMPLEX | CapsMIPMAP | CapsTEXTURE
)

// Contains reports whether all bits in o are set.
func (f Caps) Contains(o Caps) bool { return f&o == o }

// Insert sets the bits in o.
func (f *Caps) Insert(o Caps) { *f |= o }

// Bits returns the raw bit value.
func (f Caps) Bits() uint32 { return uint32(f) }

func (f Caps) String() string {
	return formatFlags(f, []flagName[Caps]{
		{CapsCOMPLEX, "COMPLEX"},
		{CapsMIPMAP, "MIPMAP"},
		{CapsTEXTURE, "TEXTURE"},
	})
}

// Caps2 provides additional detail about the surfaces stored.
type Caps2 uint32

const (
	// Caps2CUBEMAP is required for a cube map.
	Caps2CUBEMAP Caps2 = 0x200
	// Caps2CUBEMAP_POSITIVEX is required when these surfaces are stored in a
	// cubemap.
	Caps2CUBEMAP_POSITIVEX Caps2 = 0x400
	// Caps2CUBEMAP_NEGATIVEX is required when these surfaces are stored in a
	// cubemap.
	Caps2CUBEMAP_NEGATIVEX Caps2 = 0x800
	// Caps2CUBEMAP_POSITIVEY is required when these surfaces are stored in a
	// cubemap.
	Caps2CUBEMAP_POSITIVEY Caps2 = 0x1000
	// Caps2CUBEMAP_NEGATIVEY is required when these surfaces are stored in a
	// cubemap.
	Caps2CUBEMAP_NEGATIVEY Caps2 = 0x2000
	// Caps2CUBEMAP_POSITIVEZ is required when these surfaces are stored in a
	// cubemap.
	Caps2CUBEMAP_POSITIVEZ Caps2 = 0x4000
	// Caps2CUBEMAP_NEGATIVEZ is required when these surfaces are stored in a
	// cubemap.
	Caps2CUBEMAP_NEGATIVEZ Caps2 = 0x8000
	// Caps2VOLUME is required for a volume texture.
	Caps2VOLUME Caps2 = 0x200000
	// Caps2CUBEMAP_ALLFACES is identical to setting all cubemap direction
	// flags.
	Caps2CUBEMAP_ALLFACES Caps2 = Caps2CUBEMAP_POSITIVEX | Caps2CUBEMAP_NEGATIVEX |
		Caps2CUBEMAP_POSITIVEY | Caps2CUBEMAP_NEGATIVEY |
		Caps2CUBEMAP_POSITIVEZ | Caps2CUBEMAP_NEGATIVEZ

	caps2All Caps2 = Caps2CUBEMAP | Caps2CUBEMAP_ALLFACES | Caps2VOLUME
)

// Contains reports whether all bits in o are set.
func (f Caps2) Contains(o Caps2) bool { return f&o == o }

// Insert sets the bits in o.
func (f *Caps2) Insert(o Caps2) { *f |= o }

// Bits returns the raw bit value.
func (f Caps2) Bits() uint32 { return uint32(f) }

func (f Caps2) String() string {
	return formatFlags(f, []flagName[Caps2]{
		{Caps2CUBEMAP, "CUBEMAP"},
		{Caps2CUBEMAP_POSITIVEX, "CUBEMAP_POSITIVEX"},
		{Caps2CUBEMAP_NEGATIVEX, "CUBEMAP_NEGATIVEX"},
		{Caps2CUBEMAP_POSITIVEY, "CUBEMAP_POSITIVEY"},
		{Caps2CUBEMAP_NEGATIVEY, "CUBEMAP_NEGATIVEY"},
		{Caps2CUBEMAP_POSITIVEZ, "CUBEMAP_POSITIVEZ"},
		{Caps2CUBEMAP_NEGATIVEZ, "CUBEMAP_NEGATIVEZ"},
		{Caps2VOLUME, "VOLUME"},
	})
}

// Header is the main DDS header (the 124-byte structure following the magic).
type Header struct {
	// Size of this structure in bytes; set to 124.
	Size uint32

	// Flags indicating which members contain valid data.
	Flags HeaderFlags

	// Surface height (in pixels).
	Height uint32

	// Surface width (in pixels).
	Width uint32

	// The pitch or number of bytes per scan line in an uncompressed texture.
	Pitch *uint32

	// The total number of bytes in a top level texture for a compressed
	// texture.
	LinearSize *uint32

	// Depth of a volume texture (in pixels).
	Depth *uint32

	// Number of mipmap levels.
	MipMapCount *uint32

	// Unused (reserved); we write back what we read.
	Reserved1 [11]uint32

	// The pixel format.
	SPF PixelFormat

	// Specifies the complexity of the surfaces stored.
	Caps Caps

	// Additional detail about the surfaces stored.
	Caps2 Caps2

	// Unused; we write back what we read.
	Caps3 uint32

	// Unused; we write back what we read.
	Caps4 uint32

	// Unused; we write back what we read.
	Reserved2 uint32
}

// DefaultHeader returns a header populated with conventional default values.
func DefaultHeader() Header {
	return Header{
		Size:  124, // must be 124
		Flags: HeaderFlagsCAPS | HeaderFlagsHEIGHT | HeaderFlagsWIDTH | HeaderFlagsPIXELFORMAT,
		Caps:  CapsTEXTURE,
	}
}

// NewHeaderD3D creates a header for a new DDS with a D3DFormat.
func NewHeaderD3D(
	height uint32,
	width uint32,
	depth *uint32,
	format D3DFormat,
	mipmapLevels *uint32,
	caps2 *Caps2,
) (Header, error) {
	header := DefaultHeader()
	header.Height = height
	header.Width = width
	header.MipMapCount = mipmapLevels
	header.Depth = depth
	header.SPF = PixelFormatFromD3D(format)

	if mipmapLevels != nil && *mipmapLevels > 1 {
		header.Flags.Insert(HeaderFlagsMIPMAPCOUNT)
		header.Caps.Insert(CapsCOMPLEX | CapsMIPMAP)
	}
	if depth != nil && *depth > 1 {
		header.Caps.Insert(CapsCOMPLEX)
		header.Flags.Insert(HeaderFlagsDEPTH)
	}

	// Let the caller handle caps2.
	if caps2 != nil {
		header.Caps2 = *caps2
	}

	_, compressed := format.GetBlockSize()
	pitch, ok := format.GetPitch(width)
	if !ok {
		return header, ErrUnsupportedFormat
	}

	d := u32Val(depth, 1)

	if compressed {
		header.Flags.Insert(HeaderFlagsLINEARSIZE)
		pitchHeight := format.GetPitchHeight()
		rawHeight := (height + (pitchHeight - 1)) / pitchHeight
		header.LinearSize = u32p(pitch * rawHeight * d)
	} else {
		header.Flags.Insert(HeaderFlagsPITCH)
		header.Pitch = u32p(pitch)
	}

	return header, nil
}

// NewHeaderDXGI creates a header for a new DDS with a DxgiFormat.
func NewHeaderDXGI(
	height uint32,
	width uint32,
	depth *uint32,
	format DxgiFormat,
	mipmapLevels *uint32,
	arrayLayers *uint32,
	caps2 *Caps2,
) (Header, error) {
	header := DefaultHeader()
	header.Height = height
	header.Width = width
	header.MipMapCount = mipmapLevels
	header.Depth = depth
	header.SPF = PixelFormatFromDXGI(format)

	if mipmapLevels != nil && *mipmapLevels > 1 {
		header.Flags.Insert(HeaderFlagsMIPMAPCOUNT)
		header.Caps.Insert(CapsCOMPLEX | CapsMIPMAP)
	}
	if depth != nil && *depth > 1 {
		header.Caps.Insert(CapsCOMPLEX)
		header.Flags.Insert(HeaderFlagsDEPTH)
	}
	if arrayLayers != nil && *arrayLayers > 1 {
		header.Caps.Insert(CapsCOMPLEX)
	}

	// Let the caller handle caps2.
	if caps2 != nil {
		header.Caps2 = *caps2
	}

	_, compressed := format.GetBlockSize()
	pitch, ok := format.GetPitch(width)
	if !ok {
		return header, ErrUnsupportedFormat
	}

	d := u32Val(depth, 1)

	if compressed {
		header.Flags.Insert(HeaderFlagsLINEARSIZE)
		pitchHeight := format.GetPitchHeight()
		rawHeight := (height + (pitchHeight - 1)) / pitchHeight
		header.LinearSize = u32p(pitch * rawHeight * d)
	} else {
		header.Flags.Insert(HeaderFlagsPITCH)
		header.Pitch = u32p(pitch)
	}

	return header, nil
}

// ReadHeader reads a Header structure from r.
func ReadHeader(r io.Reader) (Header, error) {
	var h Header
	size, err := readU32(r)
	if err != nil {
		return h, err
	}
	if size != 124 {
		return h, InvalidFieldError("Header struct size")
	}
	flagsRaw, err := readU32(r)
	if err != nil {
		return h, err
	}
	height, err := readU32(r)
	if err != nil {
		return h, err
	}
	width, err := readU32(r)
	if err != nil {
		return h, err
	}
	pitchOrLinearSize, err := readU32(r)
	if err != nil {
		return h, err
	}
	depth, err := readU32(r)
	if err != nil {
		return h, err
	}
	mipMapCount, err := readU32(r)
	if err != nil {
		return h, err
	}
	var reserved1 [11]uint32
	for i := range reserved1 {
		v, err := readU32(r)
		if err != nil {
			return h, err
		}
		reserved1[i] = v
	}
	spf, err := ReadPixelFormat(r)
	if err != nil {
		return h, err
	}
	caps, err := readU32(r)
	if err != nil {
		return h, err
	}
	caps2, err := readU32(r)
	if err != nil {
		return h, err
	}
	caps3, err := readU32(r)
	if err != nil {
		return h, err
	}
	caps4, err := readU32(r)
	if err != nil {
		return h, err
	}
	reserved2, err := readU32(r)
	if err != nil {
		return h, err
	}

	flags := HeaderFlags(flagsRaw) & headerFlagsAll
	h = Header{
		Size:      size,
		Flags:     flags,
		Height:    height,
		Width:     width,
		Reserved1: reserved1,
		SPF:       spf,
		Caps:      Caps(caps) & capsAll,
		Caps2:     Caps2(caps2) & caps2All,
		Caps3:     caps3,
		Caps4:     caps4,
		Reserved2: reserved2,
	}
	if flags.Contains(HeaderFlagsPITCH) {
		h.Pitch = u32p(pitchOrLinearSize)
	}
	if flags.Contains(HeaderFlagsLINEARSIZE) {
		h.LinearSize = u32p(pitchOrLinearSize)
	}
	if flags.Contains(HeaderFlagsDEPTH) {
		h.Depth = u32p(depth)
	}
	if flags.Contains(HeaderFlagsMIPMAPCOUNT) {
		h.MipMapCount = u32p(mipMapCount)
	}
	return h, nil
}

// Write writes the Header structure to w.
func (h *Header) Write(w io.Writer) error {
	if err := writeU32(w, h.Size); err != nil {
		return err
	}
	if err := writeU32(w, h.Flags.Bits()); err != nil {
		return err
	}
	if err := writeU32(w, h.Height); err != nil {
		return err
	}
	if err := writeU32(w, h.Width); err != nil {
		return err
	}
	if h.Pitch != nil {
		if err := writeU32(w, *h.Pitch); err != nil {
			return err
		}
	} else if h.LinearSize != nil {
		if err := writeU32(w, *h.LinearSize); err != nil {
			return err
		}
	} else {
		if err := writeU32(w, 0); err != nil {
			return err
		}
	}
	if err := writeU32(w, u32Val(h.Depth, 0)); err != nil {
		return err
	}
	if err := writeU32(w, u32Val(h.MipMapCount, 0)); err != nil {
		return err
	}
	for i := range h.Reserved1 {
		if err := writeU32(w, h.Reserved1[i]); err != nil {
			return err
		}
	}
	if err := h.SPF.Write(w); err != nil {
		return err
	}
	if err := writeU32(w, h.Caps.Bits()); err != nil {
		return err
	}
	if err := writeU32(w, h.Caps2.Bits()); err != nil {
		return err
	}
	if err := writeU32(w, h.Caps3); err != nil {
		return err
	}
	if err := writeU32(w, h.Caps4); err != nil {
		return err
	}
	if err := writeU32(w, h.Reserved2); err != nil {
		return err
	}
	return nil
}

func (h *Header) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "  Header:\n")
	fmt.Fprintf(&b, "    flags: %s\n", h.Flags)
	fmt.Fprintf(&b, "    height: %d, width: %d, depth: %s\n", h.Height, h.Width, optU32(h.Depth))
	fmt.Fprintf(&b, "    pitch: %s  linear_size: %s\n", optU32(h.Pitch), optU32(h.LinearSize))
	fmt.Fprintf(&b, "    mipmap_count: %s\n", optU32(h.MipMapCount))
	fmt.Fprintf(&b, "    caps: %s, caps2 %s\n", h.Caps, h.Caps2)
	b.WriteString(h.SPF.String())
	return b.String()
}
