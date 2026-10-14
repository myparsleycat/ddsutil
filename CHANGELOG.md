# Changelog

## Unreleased

### Breaking changes

The API now follows Go naming conventions. Update callers as follows:

| Before | After |
| --- | --- |
| `dds.GetWidth()`, `GetHeight()`, `GetDepth()` | `dds.Width()`, `Height()`, `Depth()` |
| `dds.GetD3DFormat()`, `GetDXGIFormat()`, `GetFormat()` | `dds.D3DFormat()`, `DxgiFormat()`, `Format()` |
| `dds.GetBitsPerPixel()`, `GetPitch()`, `GetPitchHeight()` | `dds.BitsPerPixel()`, `Pitch()`, `PitchHeight()` |
| `dds.GetMainTextureSize()` (`uint32`) | `dds.MainTextureSize()` (`uint64`) |
| `dds.GetArrayStride()` (`uint32`) | `dds.ArrayStride()` (`uint64`) |
| `dds.GetNumArrayLayers()`, `GetNumMipmapLevels()` | `dds.NumArrayLayers()`, `NumMipmapLevels()` |
| `dds.GetMinMipmapSizeInBytes()` | `dds.MinMipmapSizeInBytes()` |
| `dds.GetData(layer)` | `dds.LayerData(layer)` |
| `ddsutil.DDSImageFormat(dds)` | `dds.ImageFormat()` |
| `DataFormat.GetPitch`, `GetPitchHeight`, `GetBitsPerPixel`, `GetBlockSize`, `GetFourCC`, `GetMinimumMipmapSizeInBytes` | `Pitch`, `PitchHeight`, `BitsPerPixel`, `BlockSize`, `FourCC`, `MinimumMipmapSizeInBytes` |
| `NewDXGI(NewDxgiParams{...})` | `NewDxgi(DxgiParams{...})` |
| `NewD3D(NewD3dParams{...})` | `NewD3D(D3DParams{...})` |
| `NewHeaderD3D(height, width, depth, format, mips, caps2)` | `NewHeaderD3D(D3DParams{...})` |
| `NewHeaderDXGI(height, width, depth, format, mips, layers, caps2)` | `NewHeaderDxgi(DxgiParams{...})` |
| `SurfaceRgba8.GetImage(layer, depth, mip)` | `SurfaceRgba8.Image(layer, depth, mip)` |
| `GeneratedExact(n)`, `Mipmaps.GeneratedExactCount()` | `MipmapsGeneratedExact(n)` |

- The optional `*uint32` and `*Caps2` fields of the parameter structs are now
  plain values where 0 selects the default: `Depth: u32p(2)` becomes
  `Depth: 2`, and `MipmapLevels`, `ArrayLayers` and `Caps2` work the same way.
- `Mipmaps` is now a struct. `MipmapsDisabled`, `MipmapsFromSurface` and
  `MipmapsGeneratedAutomatic` are values of it, its zero value is
  `MipmapsDisabled`, and the old `MipmapsGeneratedExact` constant is replaced
  by the `MipmapsGeneratedExact(count)` function.
- Encoding with more mipmaps than the surface dimensions allow returns a
  `SurfaceErrorUnexpectedMipmapCount` error instead of producing a surface
  that fails validation.
- The block compressors' `Quality` levels now produce different (better or
  equal) output, see below.

### Added

- `ErrTooLarge`, returned when a size computation would overflow.
- `ErrInvalidField` and `ErrGeneral`, wrapped by `InvalidFieldError` and
  `GeneralError` so they can be tested with `errors.Is`.
- `CreateImageError` unwraps to its `SurfaceError`.
- `Mipmaps` implements `fmt.Stringer`.

### Fixed

- BC4/BC5 snorm encoding produced unsigned blocks, so decoded values were off
  by about 128.
- 3D textures with mipmaps failed to decode (`MipmapDataOutOfBounds`) and
  mixed up depth slices.
- BC6H and BC7 blocks whose first pixel was closest to the second endpoint
  decoded that pixel with the wrong color.
- `Surface.ToDds` wrote cube maps with a DX10 array size of 0, which is
  invalid and rejected by `NewDdsReader`.
- A header declaring 2^32-1 mipmaps made `LayerData` loop for seconds, and
  size computations wrapped around for textures over 4 GiB.

### Performance

- BCn encoding and decoding run block rows in parallel.
- `Quality` was previously ignored. `QualityFast` uses principal axis or
  value range endpoints, `QualityNormal` adds least squares refinement or a
  local search, and `QualitySlow` adds exhaustive searches. `QualityNormal`
  for a 1024x1024 texture on 4 cores went from 697ms to 29ms for BC1 and from
  4.7s to 208ms for BC3.
