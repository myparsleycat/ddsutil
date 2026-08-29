# ddsutil
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fmyparsleycat%2Fddsutil.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fmyparsleycat%2Fddsutil?ref=badge_shield)


A unified DirectDraw Surface (DDS) library for Go, providing two layers in a
single package:

| Layer | Role |
| --- | --- |
| File container | DDS header/DX10 header parsing & writing, `D3DFormat`/`DxgiFormat` enums, pixel format structures |
| Image codec | BC1–BC7 decode/encode, RGBA8/RGBAF32 conversion, mipmap generation, `image.RGBA` interop |

The codec layer builds directly on the container's `Dds` type, so there is one
parser, one set of format enums, and one error model. No external
dependencies (Go standard library only).

## Features

- Full DDS header parsing and writing: the 124-byte header, the optional
  20-byte `DDS_HEADER_DXT10` extension, legacy D3D and DXGI formats, array
  layers, cube maps, 3D textures, and mipmaps.
- Decode BC1–BC7 (DXT1/DXT3/DXT5, RGTC1/RGTC2, BPTC float/unorm) and all
  uncompressed formats to RGBA8 or RGBAF32 surfaces with self-contained Go
  block decoding.
- Encode RGBA8/RGBAF32 surfaces to BC1–BC7 and all uncompressed formats with
  self-contained Go encoders:
  - BC1/BC2/BC3: exhaustive endpoint search over the block's colors.
  - BC4/BC5: min/max endpoint fit.
  - BC6H: mode 11 (explicit 10.10.10 endpoints, no delta compression).
  - BC7: mode 6 (7-bit RGBA endpoints with p-bits, 4-bit indices).
- Mipmap generation (box-filtered downsampling).
- Random-access mip reads and bounded-memory preview decoding without loading
  the complete DDS payload or full-resolution RGBA image.
- Golden-file tested against reference DDS files.

## Usage

```go
import "github.com/myparsleycat/ddsutil"

// Read and decode a DDS file to an RGBA image.
data, _ := os.ReadFile("texture.dds")
dds, err := ddsutil.Parse(data)
img, err := ddsutil.ImageFromDds(dds, 0) // mip level 0

// Encode an RGBA image to BC7 with generated mipmaps.
dds2, err := ddsutil.DdsFromImage(img, ddsutil.BC7RgbaUnorm,
    ddsutil.QualityNormal, ddsutil.MipmapsGeneratedAutomatic)

// Work with the raw file container.
fmt.Println(dds.GetWidth(), dds.GetHeight())
data0, _ := dds.GetData(0) // mip chain of array layer 0
```

For large textures, use `DdsReader` to read or decode only the mip needed by
the caller:

```go
file, err := os.Open("texture.dds")
info, err := file.Stat()
reader, err := ddsutil.NewDdsReader(file, info.Size())
metadata := reader.Metadata()

// Decode array layer 0, mip level 2 without reading earlier mip payloads.
mip, err := reader.DecodeMipRgba8(0, 2)

// Decode directly to a smaller nearest-neighbour preview without allocating
// a full-resolution RGBA image.
preview, err := reader.DecodeMipRgba8To(0, 0, 2048, 2048)
```

The core codec types are `Surface`, `SurfaceRgba8`, and `SurfaceRgba32Float`.
Use `Surface.DecodeRgba8()`, `SurfaceRgba8.Encode(format, quality, mipmaps)`,
`Surface.ToDds()`, and `SurfaceFromDds(dds)` to work with surfaces directly,
and `ddsutil.Read`/`Dds.Write` for stream-based file I/O.

## Command-line tools

```sh
go run ./cmd/ddsinfo texture.dds        # dump header information
go run ./cmd/dds2png texture.dds out.png
go run ./cmd/retag texture.dds BC7_UNorm_sRGB   # re-tag DX10 format, no data conversion
```

## Known quirks

The container layer intentionally preserves these behaviors, e.g. the `L8`
format does not round-trip through `PixelFormat` detection, and for DX10
cubemaps `Header10.ArraySize` stores the number of cubemaps (array
layers / 6).

## Testing

```sh
go test ./...
```

`golden_test.go` compares container output byte-for-byte against the golden
reference files in `testdata/`.

## License

MIT, see LICENSE.


[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fmyparsleycat%2Fddsutil.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fmyparsleycat%2Fddsutil?ref=badge_large)