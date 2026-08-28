package ddsutil

import (
	"errors"
	"fmt"
	"io"
)

const (
	classicDdsPayloadOffset = uint64(128)
	dx10DdsPayloadOffset    = uint64(148)
	maxInt64Value           = uint64(1<<63 - 1)
)

// DdsMetadata describes the image layout exposed by a DdsReader.
type DdsMetadata struct {
	Width       uint32
	Height      uint32
	Depth       uint32
	Layers      uint32
	Mipmaps     uint32
	ImageFormat ImageFormat
}

// DdsReader provides random access to individual DDS mip levels without
// reading the complete payload into memory.
type DdsReader struct {
	src           io.ReaderAt
	size          uint64
	payloadOffset uint64
	dds           Dds
	metadata      DdsMetadata
	format        DataFormat
}

type ddsMipLayout struct {
	offset uint64
	size   uint64
	width  uint32
	height uint32
	depth  uint32
	pitch  uint32
}

// NewDdsReader parses DDS metadata from src. The payload remains unread until
// ReadMip or one of the decode methods is called.
func NewDdsReader(src io.ReaderAt, size int64) (*DdsReader, error) {
	if src == nil {
		return nil, GeneralError("DDS reader source is nil")
	}
	if size < 0 {
		return nil, InvalidFieldError("file size")
	}
	section := io.NewSectionReader(src, 0, size)
	magic, err := readU32(section)
	if err != nil {
		return nil, err
	}
	if magic != Magic {
		return nil, ErrBadMagicNumber
	}
	header, err := ReadHeader(section)
	if err != nil {
		return nil, err
	}
	payloadOffset := classicDdsPayloadOffset
	var header10 *Header10
	if header.SPF.FourCC != nil && *header.SPF.FourCC == FourCC(FourCCDX10) {
		header10, err = ReadHeader10(section)
		if err != nil {
			return nil, err
		}
		payloadOffset = dx10DdsPayloadOffset
	}
	if uint64(size) < payloadOffset {
		return nil, ErrShortFile
	}
	dds := Dds{Header: header, Header10: header10}
	format := dds.GetFormat()
	if format == nil {
		return nil, ErrUnsupportedFormat
	}
	imageFormat, err := DDSImageFormat(&dds)
	if err != nil {
		return nil, err
	}
	metadata := DdsMetadata{
		Width:       dds.GetWidth(),
		Height:      dds.GetHeight(),
		Depth:       dds.GetDepth(),
		Layers:      arrayLayerCount(&dds),
		Mipmaps:     dds.GetNumMipmapLevels(),
		ImageFormat: imageFormat,
	}
	if metadata.Width == 0 {
		return nil, InvalidFieldError("width")
	}
	if metadata.Height == 0 {
		return nil, InvalidFieldError("height")
	}
	if metadata.Depth == 0 {
		return nil, InvalidFieldError("depth")
	}
	if metadata.Layers == 0 {
		return nil, InvalidFieldError("array layers")
	}
	if metadata.Mipmaps == 0 {
		return nil, InvalidFieldError("mipmap count")
	}
	maxMipmaps := maxMipmapCount(max3(metadata.Width, metadata.Height, metadata.Depth))
	if metadata.Mipmaps > maxMipmaps {
		return nil, &SurfaceError{Kind: SurfaceErrorUnexpectedMipmapCount, Mipmaps: metadata.Mipmaps, MaxTotalMipmaps: maxMipmaps}
	}
	if header10 != nil {
		if header10.ArraySize == 0 {
			return nil, InvalidFieldError("array size")
		}
		if header10.ResourceDimension == D3D10ResourceDimensionTexture3D && (header10.ArraySize != 1 || header10.MiscFlag.Contains(MiscFlagTEXTURECUBE)) {
			return nil, InvalidFieldError("3D texture array layout")
		}
	}
	r := &DdsReader{
		src:           src,
		size:          uint64(size),
		payloadOffset: payloadOffset,
		dds:           dds,
		metadata:      metadata,
		format:        format,
	}
	if _, err := r.mipLayout(metadata.Layers-1, metadata.Mipmaps-1); err != nil {
		return nil, err
	}
	return r, nil
}

// Metadata returns a copy of the parsed DDS image metadata.
func (r *DdsReader) Metadata() DdsMetadata {
	if r == nil {
		return DdsMetadata{}
	}
	return r.metadata
}

// ReadMip reads one flattened array layer and mip level into a standalone
// single-layer, single-mipmap surface.
func (r *DdsReader) ReadMip(layer, mipmap uint32) (*Surface, error) {
	layout, err := r.mipLayout(layer, mipmap)
	if err != nil {
		return nil, err
	}
	length, err := uint64ToInt(layout.size)
	if err != nil {
		return nil, err
	}
	data := make([]byte, length)
	if err = r.readAt(data, layout.offset); err != nil {
		return nil, err
	}
	return &Surface{
		Width:       layout.width,
		Height:      layout.height,
		Depth:       layout.depth,
		Layers:      1,
		Mipmaps:     1,
		ImageFormat: r.metadata.ImageFormat,
		Data:        data,
	}, nil
}

// DecodeMipRgba8 reads and decodes one flattened array layer and mip level.
// A 2D mip is decoded directly into the returned buffer without a second copy.
func (r *DdsReader) DecodeMipRgba8(layer, mipmap uint32) (*SurfaceRgba8, error) {
	surface, err := r.ReadMip(layer, mipmap)
	if err != nil {
		return nil, err
	}
	if surface.Depth == 1 {
		data, decodeErr := decodeU8(surface.Width, surface.Height, surface.ImageFormat, surface.Data)
		if decodeErr != nil {
			return nil, decodeErr
		}
		return &SurfaceRgba8{Width: surface.Width, Height: surface.Height, Depth: 1, Layers: 1, Mipmaps: 1, Data: data}, nil
	}
	length, err := rgbaLength(surface.Width, surface.Height, surface.Depth)
	if err != nil {
		return nil, err
	}
	data := make([]byte, length)
	sliceLength := length / int(surface.Depth)
	for depthLevel := uint32(0); depthLevel < surface.Depth; depthLevel++ {
		compressed := surface.Get(0, depthLevel, 0)
		if compressed == nil {
			return nil, &SurfaceError{Kind: SurfaceErrorMipmapDataOutOfBounds, Layer: layer, Mipmap: mipmap}
		}
		decoded, decodeErr := decodeU8(surface.Width, surface.Height, surface.ImageFormat, compressed)
		if decodeErr != nil {
			return nil, decodeErr
		}
		copy(data[int(depthLevel)*sliceLength:], decoded)
	}
	return &SurfaceRgba8{Width: surface.Width, Height: surface.Height, Depth: surface.Depth, Layers: 1, Mipmaps: 1, Data: data}, nil
}

// DecodeMipRgba8To decodes a 2D mip directly to targetWidth x targetHeight
// using nearest-neighbour sampling without allocating a full-size RGBA image.
func (r *DdsReader) DecodeMipRgba8To(layer, mipmap, targetWidth, targetHeight uint32) (*SurfaceRgba8, error) {
	layout, err := r.mipLayout(layer, mipmap)
	if err != nil {
		return nil, err
	}
	if targetWidth == 0 || targetHeight == 0 {
		return nil, InvalidFieldError("target dimensions")
	}
	if targetWidth > layout.width || targetHeight > layout.height {
		return nil, InvalidFieldError("target dimensions exceed source mip")
	}
	if layout.depth != 1 {
		return nil, ErrUnsupportedFormat
	}
	if targetWidth == layout.width && targetHeight == layout.height {
		return r.DecodeMipRgba8(layer, mipmap)
	}
	length, err := rgbaLength(targetWidth, targetHeight, 1)
	if err != nil {
		return nil, err
	}
	data := make([]byte, length)
	if r.metadata.ImageFormat.IsBlockCompressed() {
		err = r.decodeCompressedMipTo(data, layout, targetWidth, targetHeight)
	} else {
		err = r.decodeUncompressedMipTo(data, layout, targetWidth, targetHeight)
	}
	if err != nil {
		return nil, err
	}
	return &SurfaceRgba8{Width: targetWidth, Height: targetHeight, Depth: 1, Layers: 1, Mipmaps: 1, Data: data}, nil
}

func (r *DdsReader) decodeCompressedMipTo(dst []byte, layout ddsMipLayout, targetWidth, targetHeight uint32) error {
	bc, ok := imageFormatBC(r.metadata.ImageFormat)
	if !ok {
		return ErrUnsupportedFormat
	}
	blockSize := bc.blockSizeInBytes()
	rowLength := int(layout.pitch)
	row := make([]byte, rowLength)
	currentBlockRow := ^uint32(0)
	for y := uint32(0); y < targetHeight; y++ {
		srcY := uint32(uint64(y) * uint64(layout.height) / uint64(targetHeight))
		blockRow := srcY / blockHeight
		if blockRow != currentBlockRow {
			rowOffset, ok := checkedAdd(layout.offset, uint64(blockRow)*uint64(layout.pitch))
			if !ok {
				return &SurfaceError{Kind: SurfaceErrorPixelCountWouldOverflow, Width: layout.width, Height: layout.height, Depth: 1}
			}
			if err := r.readAt(row, rowOffset); err != nil {
				return err
			}
			currentBlockRow = blockRow
		}
		currentBlockColumn := ^uint32(0)
		var pixels [blockHeight][blockWidth][4]uint8
		for x := uint32(0); x < targetWidth; x++ {
			srcX := uint32(uint64(x) * uint64(layout.width) / uint64(targetWidth))
			blockColumn := srcX / blockWidth
			if blockColumn != currentBlockColumn {
				start := int(blockColumn) * blockSize
				if start < 0 || start+blockSize > len(row) {
					return ErrShortFile
				}
				pixels = decompressBlock(bc, row[start:start+blockSize])
				currentBlockColumn = blockColumn
			}
			pixel := pixels[srcY%blockHeight][srcX%blockWidth]
			offset := (int(y)*int(targetWidth) + int(x)) * channels
			copy(dst[offset:offset+channels], pixel[:])
		}
	}
	return nil
}

func (r *DdsReader) decodeUncompressedMipTo(dst []byte, layout ddsMipLayout, targetWidth, targetHeight uint32) error {
	row := make([]byte, int(layout.pitch))
	var decoded []byte
	currentSourceRow := ^uint32(0)
	for y := uint32(0); y < targetHeight; y++ {
		srcY := uint32(uint64(y) * uint64(layout.height) / uint64(targetHeight))
		if srcY != currentSourceRow {
			rowOffset, ok := checkedAdd(layout.offset, uint64(srcY)*uint64(layout.pitch))
			if !ok {
				return &SurfaceError{Kind: SurfaceErrorPixelCountWouldOverflow, Width: layout.width, Height: layout.height, Depth: 1}
			}
			if err := r.readAt(row, rowOffset); err != nil {
				return err
			}
			var err error
			decoded, err = decodeU8(layout.width, 1, r.metadata.ImageFormat, row)
			if err != nil {
				return err
			}
			currentSourceRow = srcY
		}
		for x := uint32(0); x < targetWidth; x++ {
			srcX := uint32(uint64(x) * uint64(layout.width) / uint64(targetWidth))
			sourceOffset := int(srcX) * channels
			targetOffset := (int(y)*int(targetWidth) + int(x)) * channels
			copy(dst[targetOffset:targetOffset+channels], decoded[sourceOffset:sourceOffset+channels])
		}
	}
	return nil
}

func (r *DdsReader) mipLayout(layer, mipmap uint32) (ddsMipLayout, error) {
	if r == nil || r.src == nil || r.format == nil {
		return ddsMipLayout{}, GeneralError("DDS reader is nil")
	}
	if layer >= r.metadata.Layers || mipmap >= r.metadata.Mipmaps {
		return ddsMipLayout{}, ErrOutOfBounds
	}
	layouts := make([]ddsMipLayout, r.metadata.Mipmaps)
	layerStride := uint64(0)
	for index := uint32(0); index < r.metadata.Mipmaps; index++ {
		width := MipDimension(r.metadata.Width, index)
		height := MipDimension(r.metadata.Height, index)
		depth := MipDimension(r.metadata.Depth, index)
		pitch, ok := r.mipPitch(width)
		if !ok {
			return ddsMipLayout{}, ErrUnsupportedFormat
		}
		pitchHeight := r.format.GetPitchHeight()
		if pitchHeight == 0 {
			return ddsMipLayout{}, ErrUnsupportedFormat
		}
		rows := (uint64(height) + uint64(pitchHeight) - 1) / uint64(pitchHeight)
		size, ok := checkedMul(uint64(pitch), rows)
		if ok {
			size, ok = checkedMul(size, uint64(depth))
		}
		if !ok {
			return ddsMipLayout{}, &SurfaceError{Kind: SurfaceErrorPixelCountWouldOverflow, Width: width, Height: height, Depth: depth}
		}
		layouts[index] = ddsMipLayout{offset: layerStride, size: size, width: width, height: height, depth: depth, pitch: pitch}
		layerStride, ok = checkedAdd(layerStride, size)
		if !ok {
			return ddsMipLayout{}, &SurfaceError{Kind: SurfaceErrorPixelCountWouldOverflow, Width: width, Height: height, Depth: depth}
		}
	}
	layerOffset, ok := checkedMul(uint64(layer), layerStride)
	if !ok {
		return ddsMipLayout{}, ErrOutOfBounds
	}
	offset, ok := checkedAdd(r.payloadOffset, layerOffset)
	if ok {
		offset, ok = checkedAdd(offset, layouts[mipmap].offset)
	}
	if !ok {
		return ddsMipLayout{}, ErrOutOfBounds
	}
	end, ok := checkedAdd(offset, layouts[mipmap].size)
	if !ok || end > r.size || end > maxInt64Value {
		return ddsMipLayout{}, ErrShortFile
	}
	layout := layouts[mipmap]
	layout.offset = offset
	return layout, nil
}

func (r *DdsReader) mipPitch(width uint32) (uint32, bool) {
	pitch, ok := r.format.GetPitch(width)
	if !ok || pitch == 0 {
		return 0, false
	}
	blockSize := r.metadata.ImageFormat.BlockSizeInBytes()
	if blockSize <= 0 {
		return 0, false
	}
	blocks := uint64(width)
	if r.metadata.ImageFormat.IsBlockCompressed() {
		blocks = (blocks + blockWidth - 1) / blockWidth
	}
	expected, valid := checkedMul(blocks, uint64(blockSize))
	if !valid || expected > uint64(^uint32(0)) || uint32(expected) != pitch {
		return 0, false
	}
	return pitch, true
}

func (r *DdsReader) readAt(data []byte, offset uint64) error {
	if len(data) == 0 {
		return nil
	}
	end, ok := checkedAdd(offset, uint64(len(data)))
	if !ok || end > r.size || offset > maxInt64Value {
		return ErrShortFile
	}
	n, err := r.src.ReadAt(data, int64(offset))
	if n != len(data) {
		if err == nil {
			err = io.ErrUnexpectedEOF
		}
		return mapIO(err)
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return mapIO(err)
	}
	return nil
}

func imageFormatBC(format ImageFormat) (bcFormat, bool) {
	switch format {
	case BC1RgbaUnorm, BC1RgbaUnormSrgb:
		return bcFmt1, true
	case BC2RgbaUnorm, BC2RgbaUnormSrgb:
		return bcFmt2, true
	case BC3RgbaUnorm, BC3RgbaUnormSrgb:
		return bcFmt3, true
	case BC4RUnorm:
		return bcFmt4, true
	case BC4RSnorm:
		return bcFmt4S, true
	case BC5RgUnorm:
		return bcFmt5, true
	case BC5RgSnorm:
		return bcFmt5S, true
	case BC6hRgbUfloat, BC6hRgbSfloat:
		return bcFmt6, true
	case BC7RgbaUnorm, BC7RgbaUnormSrgb:
		return bcFmt7, true
	default:
		return 0, false
	}
}

func rgbaLength(width, height, depth uint32) (int, error) {
	size, ok := checkedMul(uint64(width), uint64(height))
	if ok {
		size, ok = checkedMul(size, uint64(depth))
	}
	if ok {
		size, ok = checkedMul(size, channels)
	}
	if !ok || size > uint64(maxIntValue()) {
		return 0, &SurfaceError{Kind: SurfaceErrorPixelCountWouldOverflow, Width: width, Height: height, Depth: depth}
	}
	return int(size), nil
}

func uint64ToInt(value uint64) (int, error) {
	if value > uint64(maxIntValue()) {
		return 0, fmt.Errorf("DDS payload size exceeds addressable memory: %w", ErrOutOfBounds)
	}
	return int(value), nil
}

func maxIntValue() int {
	return int(^uint(0) >> 1)
}

func checkedAdd(left, right uint64) (uint64, bool) {
	if ^uint64(0)-left < right {
		return 0, false
	}
	return left + right, true
}

func checkedMul(left, right uint64) (uint64, bool) {
	if left != 0 && ^uint64(0)/left < right {
		return 0, false
	}
	return left * right, true
}
