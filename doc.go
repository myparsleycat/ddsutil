// Package ddsutil is a unified DirectDraw Surface (DDS) library for Go,
// providing two layers in a single package:
//
//   - The file container layer: parsing and writing of the DDS header, the
//     DX10 extension header, pixel format structures, and the
//     D3DFormat/DxgiFormat enums.
//   - The image codec layer: decoding BC1-BC7 and all uncompressed formats to
//     RGBA8/RGBAF32 surfaces, encoding surfaces back, and mipmap generation.
//
// The codec layer builds directly on the container's Dds type, so there is
// one parser, one set of format enums, and one error model. The package has
// no external dependencies (Go standard library only).
//
// Quick start:
//
//	// Read and decode a DDS file to an RGBA image.
//	data, _ := os.ReadFile("texture.dds")
//	dds, err := ddsutil.Parse(data)
//	img, err := ddsutil.ImageFromDds(dds, 0)
//
//	// Encode an RGBA image to BC7 with generated mipmaps.
//	dds2, err := ddsutil.DdsFromImage(img, ddsutil.BC7RgbaUnorm,
//	    ddsutil.QualityNormal, ddsutil.MipmapsGeneratedAutomatic)
//
// The MIT License (MIT); see LICENSE for the full text.
package ddsutil