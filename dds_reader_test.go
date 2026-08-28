package ddsutil

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

type readerAtRange struct {
	offset int64
	length int
}

type trackingReaderAt struct {
	reader *bytes.Reader
	ranges []readerAtRange
}

func (r *trackingReaderAt) ReadAt(data []byte, offset int64) (int, error) {
	r.ranges = append(r.ranges, readerAtRange{offset: offset, length: len(data)})
	return r.reader.ReadAt(data, offset)
}

func TestDdsReaderReadsOnlySelectedMip(t *testing.T) {
	encoded := encodeReaderFixture(t, 64, 32, 1, BC7RgbaUnormSrgb, MipmapsGeneratedAutomatic)
	raw := ddsBytes(t, encoded)
	tracked := &trackingReaderAt{reader: bytes.NewReader(raw)}
	reader, err := NewDdsReader(tracked, int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	metadata := reader.Metadata()
	if metadata.Width != 64 || metadata.Height != 32 || metadata.Depth != 1 || metadata.Layers != 1 || metadata.Mipmaps != encoded.GetNumMipmapLevels() || metadata.ImageFormat != BC7RgbaUnormSrgb {
		t.Fatalf("metadata = %#v", metadata)
	}
	tracked.ranges = nil
	mip := uint32(2)
	selected, err := reader.ReadMip(0, mip)
	if err != nil {
		t.Fatal(err)
	}
	original, err := SurfaceFromDds(encoded)
	if err != nil {
		t.Fatal(err)
	}
	want := original.Get(0, 0, mip)
	if selected.Width != 16 || selected.Height != 8 || selected.Depth != 1 || selected.Layers != 1 || selected.Mipmaps != 1 || !bytes.Equal(selected.Data, want) {
		t.Fatalf("selected mip = %#v data=%d want=%d", selected, len(selected.Data), len(want))
	}
	start := int64(148 + len(original.Get(0, 0, 0)) + len(original.Get(0, 0, 1)))
	if len(tracked.ranges) != 1 || tracked.ranges[0].offset != start || tracked.ranges[0].length != len(want) {
		t.Fatalf("payload reads = %#v, want one read at %d with %d bytes", tracked.ranges, start, len(want))
	}
}

func TestDdsReaderReadsClassicDdsMip(t *testing.T) {
	mipmaps := uint32(2)
	dds, err := NewD3D(NewD3dParams{Width: 8, Height: 8, Format: D3DFormatDXT1, MipmapLevels: &mipmaps})
	if err != nil {
		t.Fatal(err)
	}
	for index := range dds.Data {
		dds.Data[index] = byte(index)
	}
	raw := ddsBytes(t, dds)
	reader, err := NewDdsReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if reader.Metadata().ImageFormat != BC1RgbaUnorm || reader.payloadOffset != classicDdsPayloadOffset {
		t.Fatalf("metadata/offset = %#v/%d", reader.Metadata(), reader.payloadOffset)
	}
	got, err := reader.ReadMip(0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Width != 4 || got.Height != 4 || !bytes.Equal(got.Data, dds.Data[32:40]) {
		t.Fatalf("classic mip = %dx%d %#v", got.Width, got.Height, got.Data)
	}
}

func TestDdsReaderDecodeMipMatchesFullDecode(t *testing.T) {
	formats := []ImageFormat{BC1RgbaUnorm, BC3RgbaUnorm, BC5RgUnorm, BC7RgbaUnorm, Rgba8Unorm, Bgra8Unorm}
	for _, format := range formats {
		t.Run(format.String(), func(t *testing.T) {
			encoded := encodeReaderFixture(t, 17, 9, 1, format, MipmapsGeneratedAutomatic)
			raw := ddsBytes(t, encoded)
			reader, err := NewDdsReader(bytes.NewReader(raw), int64(len(raw)))
			if err != nil {
				t.Fatal(err)
			}
			mip := uint32(1)
			got, err := reader.DecodeMipRgba8(0, mip)
			if err != nil {
				t.Fatal(err)
			}
			want, err := DecodeLayersMipmapsDds(encoded, 0, 1, mip, mip+1)
			if err != nil {
				t.Fatal(err)
			}
			if got.Width != want.Width || got.Height != want.Height || !bytes.Equal(got.Data, want.Data) {
				t.Fatalf("decoded %dx%d/%d bytes, want %dx%d/%d bytes", got.Width, got.Height, len(got.Data), want.Width, want.Height, len(want.Data))
			}
		})
	}
}

func TestDdsReaderDecodeMipToMatchesNearest(t *testing.T) {
	formats := []ImageFormat{BC1RgbaUnorm, BC3RgbaUnorm, BC5RgUnorm, BC7RgbaUnorm, Rgba8Unorm, Bgra8Unorm}
	for _, format := range formats {
		t.Run(format.String(), func(t *testing.T) {
			encoded := encodeReaderFixture(t, 19, 11, 1, format, MipmapsDisabled)
			raw := ddsBytes(t, encoded)
			tracked := &trackingReaderAt{reader: bytes.NewReader(raw)}
			reader, err := NewDdsReader(tracked, int64(len(raw)))
			if err != nil {
				t.Fatal(err)
			}
			tracked.ranges = nil
			got, err := reader.DecodeMipRgba8To(0, 0, 7, 5)
			if err != nil {
				t.Fatal(err)
			}
			full, err := DecodeLayersMipmapsDds(encoded, 0, 1, 0, 1)
			if err != nil {
				t.Fatal(err)
			}
			want := nearestRgba(full.Data, full.Width, full.Height, 7, 5)
			if got.Width != 7 || got.Height != 5 || !bytes.Equal(got.Data, want) {
				t.Fatalf("scaled pixels differ for %s", format)
			}
			payloadSize := len(raw) - 148
			maxRead := 0
			for _, read := range tracked.ranges {
				maxRead = max(maxRead, read.length)
			}
			if maxRead >= payloadSize && payloadSize > 1 {
				t.Fatalf("streaming read used entire %d-byte payload: %#v", payloadSize, tracked.ranges)
			}
		})
	}
}

func TestDdsReaderSelectsArrayLayer(t *testing.T) {
	encoded := encodeReaderFixture(t, 8, 8, 2, BC7RgbaUnorm, GeneratedExact(2))
	raw := ddsBytes(t, encoded)
	reader, err := NewDdsReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if reader.Metadata().Layers != 2 {
		t.Fatalf("layers = %d", reader.Metadata().Layers)
	}
	got, err := reader.ReadMip(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	surface, err := SurfaceFromDds(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Data, surface.Get(1, 0, 1)) {
		t.Fatal("selected array-layer mip differs")
	}
}

func TestDdsReaderSelectsCubemapFace(t *testing.T) {
	const width, height, layers = uint32(8), uint32(8), uint32(6)
	input := readerFixtureSurface(width, height, 1, layers)
	encoded, err := input.Encode(BC7RgbaUnorm, QualityFast, GeneratedExact(2))
	if err != nil {
		t.Fatal(err)
	}
	mipmaps := encoded.Mipmaps
	arrayLayers := layers
	dds, err := NewDXGI(NewDxgiParams{
		Height:            height,
		Width:             width,
		Format:            DxgiFormatBC7_UNorm,
		MipmapLevels:      &mipmaps,
		ArrayLayers:       &arrayLayers,
		IsCubemap:         true,
		ResourceDimension: D3D10ResourceDimensionTexture2D,
		AlphaMode:         AlphaModeStraight,
	})
	if err != nil {
		t.Fatal(err)
	}
	dds.Data = append([]byte(nil), encoded.Data...)
	raw := ddsBytes(t, dds)
	reader, err := NewDdsReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if reader.Metadata().Layers != 6 {
		t.Fatalf("layers = %d", reader.Metadata().Layers)
	}
	got, err := reader.ReadMip(4, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Data, encoded.Get(4, 0, 1)) {
		t.Fatal("selected cubemap face differs")
	}
}

func TestDdsReaderReadsVolumeMip(t *testing.T) {
	const width, height, depth = uint32(8), uint32(8), uint32(4)
	input := readerFixtureSurface(width, height, depth, 1)
	encoded, err := input.Encode(BC7RgbaUnorm, QualityFast, GeneratedExact(2))
	if err != nil {
		t.Fatal(err)
	}
	dds, err := encoded.ToDds()
	if err != nil {
		t.Fatal(err)
	}
	raw := ddsBytes(t, dds)
	reader, err := NewDdsReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if reader.Metadata().Depth != depth {
		t.Fatalf("depth = %d", reader.Metadata().Depth)
	}
	got, err := reader.ReadMip(0, 1)
	if err != nil {
		t.Fatal(err)
	}
	var want []byte
	for depthLevel := uint32(0); depthLevel < got.Depth; depthLevel++ {
		want = append(want, encoded.Get(0, depthLevel, 1)...)
	}
	if got.Depth != 2 || !bytes.Equal(got.Data, want) {
		t.Fatalf("volume mip depth/data = %d/%d, want 2/%d", got.Depth, len(got.Data), len(want))
	}
	if _, err = reader.DecodeMipRgba8To(0, 0, 4, 4); !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("scaled volume error = %v", err)
	}
}

func TestDdsReaderRejectsInvalidRangesAndTruncation(t *testing.T) {
	encoded := encodeReaderFixture(t, 16, 16, 1, BC7RgbaUnorm, MipmapsGeneratedAutomatic)
	raw := ddsBytes(t, encoded)
	reader, err := NewDdsReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = reader.ReadMip(1, 0); !errors.Is(err, ErrOutOfBounds) {
		t.Fatalf("layer error = %v", err)
	}
	if _, err = reader.ReadMip(0, reader.Metadata().Mipmaps); !errors.Is(err, ErrOutOfBounds) {
		t.Fatalf("mipmap error = %v", err)
	}
	if _, err = NewDdsReader(bytes.NewReader(raw[:100]), 100); !errors.Is(err, ErrShortFile) {
		t.Fatalf("header error = %v", err)
	}
	if _, err = NewDdsReader(bytes.NewReader(raw[:len(raw)-1]), int64(len(raw)-1)); !errors.Is(err, ErrShortFile) {
		t.Fatalf("payload error = %v", err)
	}
	badMagic := append([]byte(nil), raw...)
	copy(badMagic[:4], "BAD!")
	if _, err = NewDdsReader(bytes.NewReader(badMagic), int64(len(badMagic))); !errors.Is(err, ErrBadMagicNumber) {
		t.Fatalf("magic error = %v", err)
	}
}

func BenchmarkDdsReaderDecodeSelectedBC7Mip(b *testing.B) {
	encoded := encodeReaderFixture(b, 1024, 1024, 1, BC7RgbaUnorm, MipmapsGeneratedAutomatic)
	raw := ddsBytes(b, encoded)
	b.ReportAllocs()
	b.SetBytes(256 * 256 * 4)
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		reader, err := NewDdsReader(bytes.NewReader(raw), int64(len(raw)))
		if err != nil {
			b.Fatal(err)
		}
		if _, err = reader.DecodeMipRgba8(0, 2); err != nil {
			b.Fatal(err)
		}
	}
}

type testFataler interface {
	Helper()
	Fatal(args ...any)
}

func encodeReaderFixture(t testFataler, width, height, layers uint32, format ImageFormat, mipmaps Mipmaps) *Dds {
	t.Helper()
	surface := readerFixtureSurface(width, height, 1, layers)
	encoded, err := surface.Encode(format, QualityFast, mipmaps)
	if err != nil {
		t.Fatal(err)
	}
	dds, err := encoded.ToDds()
	if err != nil {
		t.Fatal(err)
	}
	return dds
}

func readerFixtureSurface(width, height, depth, layers uint32) *SurfaceRgba8 {
	data := make([]byte, int(width*height*depth*layers*4))
	for layer := uint32(0); layer < layers; layer++ {
		for depthLevel := uint32(0); depthLevel < depth; depthLevel++ {
			for y := uint32(0); y < height; y++ {
				for x := uint32(0); x < width; x++ {
					offset := int(((((layer*depth)+depthLevel)*height+y)*width + x) * 4)
					data[offset] = byte(x*13 + layer*47 + depthLevel*19)
					data[offset+1] = byte(y*17 + layer*29 + depthLevel*23)
					data[offset+2] = byte((x+y)*7 + layer*11 + depthLevel*31)
					data[offset+3] = byte(64 + (x+y+layer+depthLevel)%192)
				}
			}
		}
	}
	return &SurfaceRgba8{Width: width, Height: height, Depth: depth, Layers: layers, Mipmaps: 1, Data: data}
}

func ddsBytes(t testFataler, dds *Dds) []byte {
	t.Helper()
	var output bytes.Buffer
	if err := dds.Write(&output); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func nearestRgba(source []byte, sourceWidth, sourceHeight, targetWidth, targetHeight uint32) []byte {
	target := make([]byte, int(targetWidth*targetHeight*4))
	for y := uint32(0); y < targetHeight; y++ {
		sourceY := uint32(uint64(y) * uint64(sourceHeight) / uint64(targetHeight))
		for x := uint32(0); x < targetWidth; x++ {
			sourceX := uint32(uint64(x) * uint64(sourceWidth) / uint64(targetWidth))
			sourceOffset := int((sourceY*sourceWidth + sourceX) * 4)
			targetOffset := int((y*targetWidth + x) * 4)
			copy(target[targetOffset:targetOffset+4], source[sourceOffset:sourceOffset+4])
		}
	}
	return target
}

var _ io.ReaderAt = (*trackingReaderAt)(nil)
