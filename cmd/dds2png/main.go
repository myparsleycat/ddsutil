// Command dds2png decodes one image of a DDS file to a PNG image.
//
// Usage:
//
//	dds2png [-layer n] [-mip n] [-slice n] input.dds [output.png]
//
// The output defaults to the input file name with a .png extension. Only the
// requested mip level is read from the file.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/myparsleycat/ddsutil"
)

func main() {
	layer := flag.Uint("layer", 0, "array layer (cube map faces are layers 0-5)")
	mip := flag.Uint("mip", 0, "mip level")
	slice := flag.Uint("slice", 0, "depth slice of a volume texture")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: dds2png [-layer n] [-mip n] [-slice n] input.dds [output.png]")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() < 1 || flag.NArg() > 2 {
		flag.Usage()
		os.Exit(2)
	}
	input := flag.Arg(0)
	output := strings.TrimSuffix(input, filepath.Ext(input)) + ".png"
	if flag.NArg() == 2 {
		output = flag.Arg(1)
	}

	if err := convert(input, output, uint32(*layer), uint32(*mip), uint32(*slice)); err != nil {
		fmt.Fprintf(os.Stderr, "dds2png: %v\n", err)
		os.Exit(1)
	}
}

func convert(input, output string, layer, mip, slice uint32) error {
	file, err := os.Open(input)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}

	reader, err := ddsutil.NewDdsReader(file, info.Size())
	if err != nil {
		return fmt.Errorf("%s: %w", input, err)
	}
	metadata := reader.Metadata()
	fmt.Printf("%s: %dx%dx%d, %d layers, %d mipmaps, %v\n", input,
		metadata.Width, metadata.Height, metadata.Depth, metadata.Layers, metadata.Mipmaps, metadata.ImageFormat)
	if layer >= metadata.Layers {
		return fmt.Errorf("layer %d out of range (%d layers)", layer, metadata.Layers)
	}
	if mip >= metadata.Mipmaps {
		return fmt.Errorf("mip %d out of range (%d mipmaps)", mip, metadata.Mipmaps)
	}
	if depth := ddsutil.MipDimension(metadata.Depth, mip); slice >= depth {
		return fmt.Errorf("slice %d out of range (depth %d at mip %d)", slice, depth, mip)
	}

	surface, err := reader.DecodeMipRgba8(layer, mip)
	if err != nil {
		return fmt.Errorf("%s: %w", input, err)
	}
	pixels := surface.Get(0, slice, 0)
	if pixels == nil {
		return fmt.Errorf("%s: no data for slice %d", input, slice)
	}
	// DDS files store straight alpha, which matches image.NRGBA.
	img := &image.NRGBA{
		Pix:    pixels,
		Stride: int(surface.Width) * 4,
		Rect:   image.Rect(0, 0, int(surface.Width), int(surface.Height)),
	}

	out, err := os.Create(output)
	if err != nil {
		return err
	}
	if err := png.Encode(out, img); err != nil {
		out.Close()
		return fmt.Errorf("%s: %w", output, err)
	}
	if err := out.Close(); err != nil {
		return err
	}
	fmt.Printf("wrote %s (%dx%d)\n", output, img.Rect.Dx(), img.Rect.Dy())
	return nil
}
