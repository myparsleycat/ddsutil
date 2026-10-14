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
	// Data is the texture data. The slice returned by LayerData aliases this
	// slice, so mutating it mutates the Dds.
	Data []byte
}

// D3DParams are the parameters for NewD3D and NewHeaderD3D. Zero values
// select the defaults.
type D3DParams struct {
	Height uint32
	Width  uint32
	// Depth is the depth of a volume texture, or 0 for a 2D texture.
	Depth  uint32
	Format D3DFormat
	// MipmapLevels is the number of mip levels, or 0 to omit the mip level
	// count (a single level).
	MipmapLevels uint32
	Caps2        Caps2
}

// DxgiParams are the parameters for NewDxgi and NewHeaderDxgi. Zero values
// select the defaults.
type DxgiParams struct {
	Height uint32
	Width  uint32
	// Depth is the depth of a volume texture, or 0 for a 2D texture.
	Depth  uint32
	Format DxgiFormat
	// MipmapLevels is the number of mip levels, or 0 to omit the mip level
	// count (a single level).
	MipmapLevels uint32
	// ArrayLayers is the number of array layers (6 per cubemap), or 0 for a
	// single layer.
	ArrayLayers       uint32
	Caps2             Caps2
	IsCubemap         bool
	ResourceDimension D3D10ResourceDimension
	AlphaMode         AlphaMode
}

// NewD3D creates a new DirectDraw Surface with a D3DFormat.
func NewD3D(params D3DParams) (*Dds, error) {
	if _, ok := params.Format.MinimumMipmapSizeInBytes(); !ok {
		return nil, ErrUnsupportedFormat
	}
	arrayStride, ok, overflow := layerSize(params.Format, params.Width, params.Height, params.Depth, max(params.MipmapLevels, 1))
	if overflow {
		return nil, ErrTooLarge
	}
	if !ok {
		return nil, ErrUnsupportedFormat
	}

	dataSize, err := uint64ToInt(arrayStride)
	if err != nil {
		return nil, err
	}

	header, err := NewHeaderD3D(params)
	if err != nil {
		return nil, err
	}

	return &Dds{
		Header:   header,
		Header10: nil,
		Data:     make([]byte, dataSize),
	}, nil
}

// NewDxgi creates a new DirectDraw Surface with a DxgiFormat.
func NewDxgi(params DxgiParams) (*Dds, error) {
	arraysize := max(params.ArrayLayers, 1)

	if _, ok := params.Format.MinimumMipmapSizeInBytes(); !ok {
		return nil, ErrUnsupportedFormat
	}
	arrayStride, ok, overflow := layerSize(params.Format, params.Width, params.Height, params.Depth, max(params.MipmapLevels, 1))
	if overflow {
		return nil, ErrTooLarge
	}
	if !ok {
		return nil, ErrUnsupportedFormat
	}
	totalSize, ok := checkedMul(uint64(arraysize), arrayStride)
	if !ok {
		return nil, ErrTooLarge
	}
	dataSize, err := uint64ToInt(totalSize)
	if err != nil {
		return nil, err
	}

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

	header, err := NewHeaderDxgi(params)
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

// D3DFormat attempts to get the format of this DDS, presuming it is a
// D3DFormat.
func (d *Dds) D3DFormat() (D3DFormat, bool) {
	// FIXME: some d3d formats are equivalent to some dxgi formats, but we
	// don't have a conversion between them yet. Right now we will yield
	// false if the format is dxgi, but later on we should try to convert.
	return D3DFormatTryFromPixelFormat(&d.Header.SPF)
}

// DxgiFormat attempts to get the format of this DDS, presuming it is a
// DxgiFormat.
func (d *Dds) DxgiFormat() (DxgiFormat, bool) {
	// FIXME: some d3d formats are equivalent to some dxgi formats, but we
	// don't have a conversion between them yet. Right now we will yield
	// false if the format is d3d, but later on we should try to convert.
	if d.Header10 != nil {
		return d.Header10.DxgiFormat, true
	}
	return DxgiFormatTryFromPixelFormat(&d.Header.SPF)
}

// Format gets the format of the DDS as a DataFormat interface
// (type-erasure). Returns nil if the format is unknown.
func (d *Dds) Format() DataFormat {
	if dxgi, ok := d.DxgiFormat(); ok {
		return dxgi
	}
	if d3d, ok := d.D3DFormat(); ok {
		return d3d
	}
	return nil
}

func (d *Dds) Width() uint32  { return d.Header.Width }
func (d *Dds) Height() uint32 { return d.Header.Height }

func (d *Dds) Depth() uint32 {
	return u32Val(d.Header.Depth, 1)
}

func (d *Dds) BitsPerPixel() (uint32, bool) {
	// Try format first
	if format := d.Format(); format != nil {
		if bpp, ok := format.BitsPerPixel(); ok {
			return uint32(bpp), true
		}
	}
	// Fall back to pixel_format rgb_bit_count field
	if d.Header.SPF.RGBBitCount != nil {
		return *d.Header.SPF.RGBBitCount, true
	}
	return 0, false
}

func (d *Dds) Pitch() (uint32, bool) {
	// Try format first
	if format := d.Format(); format != nil {
		if pitch, ok := format.Pitch(d.Header.Width); ok {
			return pitch, true
		}
	}
	// Then try header.pitch
	if d.Header.Pitch != nil {
		return *d.Header.Pitch, true
	}

	// Then try to calculate it ourselves
	if bpp, ok := d.BitsPerPixel(); ok {
		return (bpp*d.Width() + 7) / 8, true
	}
	return 0, false
}

func (d *Dds) PitchHeight() uint32 {
	if format := d.Format(); format != nil {
		return format.PitchHeight()
	}
	return 1
}

// MainTextureSize returns the size in bytes of the base mip level of one
// array layer, or false if it cannot be determined.
func (d *Dds) MainTextureSize() (uint64, bool) {
	var pitchPtr *uint32
	if pitch, ok := d.Pitch(); ok {
		pitchPtr = &pitch
	}
	return getTextureSize(
		pitchPtr,
		d.Header.LinearSize,
		d.PitchHeight(),
		d.Header.Height,
		d.Header.Depth,
	)
}

// ArrayStride returns the size in bytes of one array layer including all
// of its mip levels.
//
// For known formats, every mip level's size is computed from its
// dimensions. Otherwise each level is assumed to be a quarter of the
// previous one, starting from the header's pitch or linear size.
func (d *Dds) ArrayStride() (uint64, error) {
	if format := d.Format(); format != nil {
		if stride, ok, overflow := layerSize(format, d.Width(), d.Height(), d.Depth(), d.NumMipmapLevels()); ok {
			return stride, nil
		} else if overflow {
			return 0, ErrTooLarge
		}
	}
	size, ok := d.MainTextureSize()
	if !ok {
		return 0, ErrUnsupportedFormat
	}
	mml := d.NumMipmapLevels()
	minMipmapSize := d.MinMipmapSizeInBytes()
	stride, ok := getArrayStride(size, minMipmapSize, mml)
	if !ok {
		return 0, ErrTooLarge
	}
	return stride, nil
}

func (d *Dds) NumArrayLayers() uint32 {
	if d.Header10 != nil {
		return d.Header10.ArraySize
	}
	if d.Header.Caps2.Contains(Caps2CUBEMAP) {
		return 6
	}
	return 1 // just the 1 layer
}

func (d *Dds) NumMipmapLevels() uint32 {
	if d.Header.MipMapCount != nil {
		return *d.Header.MipMapCount
	}
	return 1 // just the main image
}

func (d *Dds) MinMipmapSizeInBytes() uint32 {
	if format := d.Format(); format != nil {
		if min, ok := format.MinimumMipmapSizeInBytes(); ok {
			return min
		}
	}
	if bpp, ok := d.BitsPerPixel(); ok {
		return (bpp + 7) / 8
	}
	return 1
}

// LayerData gets a reference to the data at the given arrayLayer (which should
// be 0 for textures with just one image). The returned slice aliases d.Data,
// so mutations to it are visible in the Dds.
func (d *Dds) LayerData(arrayLayer uint32) ([]byte, error) {
	offset, size, err := d.getOffsetAndSize(arrayLayer)
	if err != nil {
		return nil, err
	}
	end, ok := checkedAdd(offset, size)
	if !ok || end > uint64(len(d.Data)) {
		return nil, ErrOutOfBounds
	}
	return d.Data[offset:end], nil
}

func (d *Dds) getOffsetAndSize(arrayLayer uint32) (uint64, uint64, error) {
	// Verify request bounds
	if arrayLayer >= d.NumArrayLayers() {
		return 0, 0, ErrOutOfBounds
	}
	arrayStride, err := d.ArrayStride()
	if err != nil {
		return 0, 0, err
	}
	offset, ok := checkedMul(uint64(arrayLayer), arrayStride)
	if !ok {
		return 0, 0, ErrTooLarge
	}
	return offset, arrayStride, nil
}

func getTextureSize(pitch, linearSize *uint32, pitchHeight, height uint32, depth *uint32) (uint64, bool) {
	d := u32Val(depth, 1)

	if linearSize != nil {
		return uint64(*linearSize), true
	}
	if pitch != nil {
		rowHeight := (uint64(height) + uint64(pitchHeight) - 1) / uint64(pitchHeight)
		size, ok := checkedMul(uint64(*pitch), rowHeight)
		if !ok {
			return 0, false
		}
		size, ok = checkedMul(size, uint64(d))
		if !ok {
			return 0, false
		}
		return size, true
	}
	return 0, false
}

// layerSize returns the size in bytes of one array layer of format with the
// given base dimensions and number of mip levels. ok is false if the format
// has no pitch; overflow reports a size that does not fit in 64 bits.
//
// The mip level count comes from the file header, so the sum is computed in
// constant time once the dimensions reach 1x1x1.
func layerSize(format DataFormat, width, height, depth, mipmapLevels uint32) (size uint64, ok, overflow bool) {
	if width == 0 || height == 0 {
		return 0, true, false
	}
	pitchHeight := uint64(format.PitchHeight())
	if pitchHeight == 0 {
		return 0, false, false
	}
	minSize, _ := format.MinimumMipmapSizeInBytes()
	for mip := uint32(0); mip < mipmapLevels; mip++ {
		w := MipDimension(width, mip)
		h := MipDimension(height, mip)
		d := MipDimension(max(depth, 1), mip)
		pitch, ok := format.Pitch(w)
		if !ok {
			return 0, false, false
		}
		mipSize, ok := checkedMul(uint64(pitch), (uint64(h)+pitchHeight-1)/pitchHeight)
		if ok {
			mipSize, ok = checkedMul(mipSize, uint64(d))
		}
		if !ok {
			return 0, false, true
		}
		mipSize = max(mipSize, uint64(minSize))
		count := uint64(1)
		if w == 1 && h == 1 && d == 1 {
			// Every remaining level has the same size.
			count = uint64(mipmapLevels - mip)
		}
		levelsSize, ok := checkedMul(mipSize, count)
		if ok {
			size, ok = checkedAdd(size, levelsSize)
		}
		if !ok {
			return 0, false, true
		}
		if count > 1 {
			break
		}
	}
	return size, true, false
}

// getArrayStride sums the sizes of mipmapLevels mip levels, each a quarter of
// the previous one but at least minMipmapSize. It returns false on overflow.
//
// The mip level count comes from the file header, so the sum is computed in
// constant time once the mip size reaches its minimum.
func getArrayStride(textureSize uint64, minMipmapSize, mipmapLevels uint32) (uint64, bool) {
	var stride uint64
	currentMipsize := textureSize
	for i := uint32(0); i < mipmapLevels; i++ {
		if currentMipsize == uint64(minMipmapSize) {
			// Every remaining level has the minimum size.
			rest, ok := checkedMul(uint64(mipmapLevels-i), currentMipsize)
			if !ok {
				return 0, false
			}
			return checkedAdd(stride, rest)
		}
		var ok bool
		if stride, ok = checkedAdd(stride, currentMipsize); !ok {
			return 0, false
		}
		currentMipsize /= 4
		if currentMipsize < uint64(minMipmapSize) {
			currentMipsize = uint64(minMipmapSize)
		}
	}
	return stride, true
}

func (d *Dds) String() string {
	var b strings.Builder
	fmt.Fprintln(&b, "Dds:")
	if d3dformat, ok := d.D3DFormat(); ok {
		fmt.Fprintf(&b, "  Format: %s\n", d3dformat)
	} else if dxgiformat, ok := d.DxgiFormat(); ok {
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
