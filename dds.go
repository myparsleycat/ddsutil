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

// Magic is the DDS magic number: b"DDS " in little endian.
const Magic uint32 = 0x20534444

// Dds is the main DirectDraw Surface file structure.
type Dds struct {
	// Header is the main DDS header.
	Header Header
	// Header10 is the optional DX10 extension header.
	Header10 *Header10
	// Data is the texture data. The slice returned by GetData aliases this
	// slice, so mutating it mutates the Dds.
	Data []byte
}

// NewD3dParams are the parameters for NewD3D.
type NewD3dParams struct {
	Height       uint32
	Width        uint32
	Depth        *uint32
	Format       D3DFormat
	MipmapLevels *uint32
	Caps2        *Caps2
}

// NewDxgiParams are the parameters for NewDXGI.
type NewDxgiParams struct {
	Height            uint32
	Width             uint32
	Depth             *uint32
	Format            DxgiFormat
	MipmapLevels      *uint32
	ArrayLayers       *uint32
	Caps2             *Caps2
	IsCubemap         bool
	ResourceDimension D3D10ResourceDimension
	AlphaMode         AlphaMode
}

// NewD3D creates a new DirectDraw Surface with a D3DFormat.
func NewD3D(params NewD3dParams) (*Dds, error) {
	pitch, ok := params.Format.GetPitch(params.Width)
	if !ok {
		return nil, ErrUnsupportedFormat
	}
	size, ok := getTextureSize(&pitch, nil, params.Format.GetPitchHeight(), params.Height, params.Depth)
	if !ok {
		return nil, ErrUnsupportedFormat
	}

	mml := u32Val(params.MipmapLevels, 1)
	minMipmapSize, ok := params.Format.GetMinimumMipmapSizeInBytes()
	if !ok {
		return nil, ErrUnsupportedFormat
	}
	arrayStride := getArrayStride(size, minMipmapSize, mml)

	dataSize := arrayStride

	header, err := NewHeaderD3D(
		params.Height,
		params.Width,
		params.Depth,
		params.Format,
		params.MipmapLevels,
		params.Caps2,
	)
	if err != nil {
		return nil, err
	}

	return &Dds{
		Header:   header,
		Header10: nil,
		Data:     make([]byte, dataSize),
	}, nil
}

// NewDXGI creates a new DirectDraw Surface with a DxgiFormat.
func NewDXGI(params NewDxgiParams) (*Dds, error) {
	arraysize := u32Val(params.ArrayLayers, 1)

	pitch, ok := params.Format.GetPitch(params.Width)
	if !ok {
		return nil, ErrUnsupportedFormat
	}
	size, ok := getTextureSize(&pitch, nil, params.Format.GetPitchHeight(), params.Height, params.Depth)
	if !ok {
		return nil, ErrUnsupportedFormat
	}

	mml := u32Val(params.MipmapLevels, 1)
	minMipmapSize, ok := params.Format.GetMinimumMipmapSizeInBytes()
	if !ok {
		return nil, ErrUnsupportedFormat
	}
	arrayStride := getArrayStride(size, minMipmapSize, mml)

	dataSize := arraysize * arrayStride

	if params.IsCubemap {
		arraysize /= 6
	}
	header10 := NewHeader10(
		params.Format,
		params.IsCubemap,
		params.ResourceDimension,
		arraysize,
		params.AlphaMode,
	)

	header, err := NewHeaderDXGI(
		params.Height,
		params.Width,
		params.Depth,
		params.Format,
		params.MipmapLevels,
		params.ArrayLayers,
		params.Caps2,
	)
	if err != nil {
		return nil, err
	}

	return &Dds{
		Header:   header,
		Header10: &header10,
		Data:     make([]byte, dataSize),
	}, nil
}

// Read reads a DDS file from r.
func Read(r io.Reader) (*Dds, error) {
	magic, err := readU32(r)
	if err != nil {
		return nil, err
	}
	if magic != Magic {
		return nil, ErrBadMagicNumber
	}

	header, err := ReadHeader(r)
	if err != nil {
		return nil, err
	}

	var header10 *Header10
	if header.SPF.FourCC != nil && *header.SPF.FourCC == FourCC(FourCCDX10) {
		header10, err = ReadHeader10(r)
		if err != nil {
			return nil, err
		}
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return &Dds{
		Header:   header,
		Header10: header10,
		Data:     data,
	}, nil
}

// Write writes the DDS file to w.
func (d *Dds) Write(w io.Writer) error {
	if err := writeU32(w, Magic); err != nil {
		return err
	}
	if err := d.Header.Write(w); err != nil {
		return err
	}
	if d.Header10 != nil {
		if err := d.Header10.Write(w); err != nil {
			return err
		}
	}
	_, err := w.Write(d.Data)
	return mapIO(err)
}

// GetD3DFormat attempts to get the format of this DDS, presuming it is a
// D3DFormat.
func (d *Dds) GetD3DFormat() (D3DFormat, bool) {
	// FIXME: some d3d formats are equivalent to some dxgi formats, but we
	// don't have a conversion between them yet. Right now we will yield
	// false if the format is dxgi, but later on we should try to convert.
	return D3DFormatTryFromPixelFormat(&d.Header.SPF)
}

// GetDXGIFormat attempts to get the format of this DDS, presuming it is a
// DxgiFormat.
func (d *Dds) GetDXGIFormat() (DxgiFormat, bool) {
	// FIXME: some d3d formats are equivalent to some dxgi formats, but we
	// don't have a conversion between them yet. Right now we will yield
	// false if the format is d3d, but later on we should try to convert.
	if d.Header10 != nil {
		return d.Header10.DxgiFormat, true
	}
	return DxgiFormatTryFromPixelFormat(&d.Header.SPF)
}

// GetFormat gets the format of the DDS as a DataFormat interface
// (type-erasure). Returns nil if the format is unknown.
func (d *Dds) GetFormat() DataFormat {
	if dxgi, ok := d.GetDXGIFormat(); ok {
		return dxgi
	}
	if d3d, ok := d.GetD3DFormat(); ok {
		return d3d
	}
	return nil
}

func (d *Dds) GetWidth() uint32  { return d.Header.Width }
func (d *Dds) GetHeight() uint32 { return d.Header.Height }

func (d *Dds) GetDepth() uint32 {
	return u32Val(d.Header.Depth, 1)
}

func (d *Dds) GetBitsPerPixel() (uint32, bool) {
	// Try format first
	if format := d.GetFormat(); format != nil {
		if bpp, ok := format.GetBitsPerPixel(); ok {
			return uint32(bpp), true
		}
	}
	// Fall back to pixel_format rgb_bit_count field
	if d.Header.SPF.RGBBitCount != nil {
		return *d.Header.SPF.RGBBitCount, true
	}
	return 0, false
}

func (d *Dds) GetPitch() (uint32, bool) {
	// Try format first
	if format := d.GetFormat(); format != nil {
		if pitch, ok := format.GetPitch(d.Header.Width); ok {
			return pitch, true
		}
	}
	// Then try header.pitch
	if d.Header.Pitch != nil {
		return *d.Header.Pitch, true
	}

	// Then try to calculate it ourselves
	if bpp, ok := d.GetBitsPerPixel(); ok {
		return (bpp*d.GetWidth() + 7) / 8, true
	}
	return 0, false
}

func (d *Dds) GetPitchHeight() uint32 {
	if format := d.GetFormat(); format != nil {
		return format.GetPitchHeight()
	}
	return 1
}

func (d *Dds) GetMainTextureSize() (uint32, bool) {
	var pitchPtr *uint32
	if pitch, ok := d.GetPitch(); ok {
		pitchPtr = &pitch
	}
	return getTextureSize(
		pitchPtr,
		d.Header.LinearSize,
		d.GetPitchHeight(),
		d.Header.Height,
		d.Header.Depth,
	)
}

func (d *Dds) GetArrayStride() (uint32, error) {
	size, ok := d.GetMainTextureSize()
	if !ok {
		return 0, ErrUnsupportedFormat
	}
	mml := d.GetNumMipmapLevels()
	minMipmapSize := d.GetMinMipmapSizeInBytes()
	return getArrayStride(size, minMipmapSize, mml), nil
}

func (d *Dds) GetNumArrayLayers() uint32 {
	if d.Header10 != nil {
		return d.Header10.ArraySize
	}
	if d.Header.Caps2.Contains(Caps2CUBEMAP) {
		return 6
	}
	return 1 // just the 1 layer
}

func (d *Dds) GetNumMipmapLevels() uint32 {
	if d.Header.MipMapCount != nil {
		return *d.Header.MipMapCount
	}
	return 1 // just the main image
}

func (d *Dds) GetMinMipmapSizeInBytes() uint32 {
	if format := d.GetFormat(); format != nil {
		if min, ok := format.GetMinimumMipmapSizeInBytes(); ok {
			return min
		}
	}
	if bpp, ok := d.GetBitsPerPixel(); ok {
		return (bpp + 7) / 8
	}
	return 1
}

// GetData gets a reference to the data at the given arrayLayer (which should
// be 0 for textures with just one image). The returned slice aliases d.Data,
// so mutations to it are visible in the Dds.
func (d *Dds) GetData(arrayLayer uint32) ([]byte, error) {
	offset, size, err := d.getOffsetAndSize(arrayLayer)
	if err != nil {
		return nil, err
	}
	end := uint64(offset) + uint64(size)
	if end > uint64(len(d.Data)) {
		return nil, ErrOutOfBounds
	}
	return d.Data[offset : offset+size], nil
}

func (d *Dds) getOffsetAndSize(arrayLayer uint32) (uint32, uint32, error) {
	// Verify request bounds
	if arrayLayer >= d.GetNumArrayLayers() {
		return 0, 0, ErrOutOfBounds
	}
	arrayStride, err := d.GetArrayStride()
	if err != nil {
		return 0, 0, err
	}
	offset := arrayLayer * arrayStride
	return offset, arrayStride, nil
}

func getTextureSize(pitch, linearSize *uint32, pitchHeight, height uint32, depth *uint32) (uint32, bool) {
	d := u32Val(depth, 1)

	if linearSize != nil {
		return *linearSize, true
	}
	if pitch != nil {
		rowHeight := (height + (pitchHeight - 1)) / pitchHeight
		return *pitch * rowHeight * d, true
	}
	return 0, false
}

func getArrayStride(textureSize, minMipmapSize, mipmapLevels uint32) uint32 {
	var stride uint32
	currentMipsize := textureSize
	for i := uint32(0); i < mipmapLevels; i++ {
		stride += currentMipsize
		currentMipsize /= 4
		if currentMipsize < minMipmapSize {
			currentMipsize = minMipmapSize
		}
	}
	return stride
}

func (d *Dds) String() string {
	var b strings.Builder
	fmt.Fprintln(&b, "Dds:")
	if d3dformat, ok := d.GetD3DFormat(); ok {
		fmt.Fprintf(&b, "  Format: %s\n", d3dformat)
	} else if dxgiformat, ok := d.GetDXGIFormat(); ok {
		fmt.Fprintf(&b, "  Format: %s\n", dxgiformat)
	} else if d.Header.SPF.FourCC != nil {
		fmt.Fprintf(&b, "  Format: FOURCC=%s (Unknown)\n", *d.Header.SPF.FourCC)
	} else {
		fmt.Fprintln(&b, "  Format UNSPECIFIED")
	}
	b.WriteString(d.Header.String())
	if d.Header10 != nil {
		b.WriteString(d.Header10.String())
	}
	fmt.Fprintln(&b, "  (data elided)")
	return b.String()
}
