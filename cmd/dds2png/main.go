// Command dds2png decodes a DDS file to a PNG image.
//
// Usage:
//
//	dds2png input.dds [output.png]
package main

import (
	"fmt"
	"image/png"
	"os"

	"github.com/myparsleycat/ddsutil"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: dds2png input.dds [output.png]")
		os.Exit(1)
	}
	input := os.Args[1]
	output := "output.png"
	if len(os.Args) > 2 {
		output = os.Args[2]
	}

	data, err := os.ReadFile(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read: %v\n", err)
		os.Exit(1)
	}

	dds, err := ddsutil.Parse(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse: %v\n", err)
		os.Exit(1)
	}

	format, err := ddsutil.DDSImageFormat(dds)
	if err != nil {
		fmt.Fprintf(os.Stderr, "format: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("dimensions: %dx%d, mipmaps: %d, layers: %d, format: %s\n",
		dds.GetWidth(), dds.GetHeight(), dds.GetNumMipmapLevels(), dds.GetNumArrayLayers(), format)

	img, err := ddsutil.ImageFromDds(dds, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "decode: %v\n", err)
		os.Exit(1)
	}

	f, err := os.Create(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintf(os.Stderr, "encode png: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (%dx%d)\n", output, img.Rect.Dx(), img.Rect.Dy())
}
