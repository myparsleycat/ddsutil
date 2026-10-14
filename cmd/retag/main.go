// Command retag changes the DXGI format recorded in a DDS file without
// converting the data, for files that are tagged with the wrong format
// (e.g. BC7_UNorm data that should be read as BC7_UNorm_sRGB).
//
// Usage:
//
//	retag [-o output.dds] [-force] input.dds format
//
// The format is a DXGI format name such as BC7_UNorm_sRGB or
// DXGI_FORMAT_BC7_UNORM_SRGB, ignoring case. Files without a DX10 header
// gain one. The input is replaced unless -o is given. Formats whose data
// size differs from the current format are rejected unless -force is given.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/myparsleycat/ddsutil"
)

func main() {
	output := flag.String("o", "", "output file (default: replace the input)")
	force := flag.Bool("force", false, "retag even if the data size does not match the new format")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: retag [-o output.dds] [-force] input.dds format")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(2)
	}
	input := flag.Arg(0)
	if *output == "" {
		*output = input
	}

	if err := retag(input, *output, flag.Arg(1), *force); err != nil {
		fmt.Fprintf(os.Stderr, "retag: %v\n", err)
		os.Exit(1)
	}
}

func retag(input, output, formatName string, force bool) error {
	format, ok := ddsutil.ParseDxgiFormat(formatName)
	if !ok {
		return fmt.Errorf("unknown DXGI format %q", formatName)
	}

	data, err := os.ReadFile(input)
	if err != nil {
		return err
	}
	dds, err := ddsutil.Parse(data)
	if err != nil {
		return fmt.Errorf("%s: %w", input, err)
	}

	oldFormat := dds.Format()

	// Retagging only makes sense when the data layout stays the same. The
	// header's pitch or linear size describes the old format, so compute
	// the sizes from the formats.
	if !force {
		if oldFormat == nil {
			return fmt.Errorf("%s: unknown current format, cannot compare data sizes (use -force to retag anyway)", input)
		}
		oldSize, oldOK := layerSize(oldFormat, dds)
		newSize, newOK := layerSize(format, dds)
		switch {
		case !oldOK || !newOK:
			return fmt.Errorf("cannot compare data sizes of %v and %v (use -force to retag anyway)", oldFormat, format)
		case oldSize != newSize:
			return fmt.Errorf("%v needs %d bytes per layer but %v has %d (use -force to retag anyway)",
				format, newSize, oldFormat, oldSize)
		}
	}

	if dds.Header10 == nil {
		addHeader10(dds, format)
	} else {
		dds.Header10.DxgiFormat = format
	}
	setPitchOrLinearSize(dds, format)

	if err := writeFileAtomic(output, dds); err != nil {
		return err
	}
	fmt.Printf("%s: %v -> %v\n", output, oldFormat, format)
	return nil
}

// layerSize returns the size in bytes of one array layer of dds stored in
// format. Mip levels past the 32nd are 1x1x1 like the 32nd, so they are not
// counted: they match between two formats whenever the 32nd does.
func layerSize(format ddsutil.DataFormat, dds *ddsutil.Dds) (uint64, bool) {
	var total uint64
	pitchHeight := format.PitchHeight()
	for mip := uint32(0); mip < dds.NumMipmapLevels() && mip < 32; mip++ {
		width := ddsutil.MipDimension(dds.Width(), mip)
		height := ddsutil.MipDimension(dds.Height(), mip)
		depth := ddsutil.MipDimension(dds.Depth(), mip)
		pitch, ok := format.Pitch(width)
		if !ok {
			return 0, false
		}
		rows := (uint64(height) + uint64(pitchHeight) - 1) / uint64(pitchHeight)
		total += uint64(pitch) * rows * uint64(depth)
	}
	return total, true
}

// addHeader10 converts a legacy header to one with a DX10 extension header.
func addHeader10(dds *ddsutil.Dds, format ddsutil.DxgiFormat) {
	isCubemap := dds.Header.Caps2.Contains(ddsutil.Caps2CUBEMAP)
	dimension := ddsutil.D3D10ResourceDimensionTexture2D
	if dds.Depth() > 1 {
		dimension = ddsutil.D3D10ResourceDimensionTexture3D
	}
	header10 := ddsutil.NewHeader10(format, isCubemap, dimension, 1, ddsutil.AlphaModeUnknown)
	dds.Header10 = &header10
	dds.Header.SPF = ddsutil.PixelFormatFromDXGI(format)
}

// setPitchOrLinearSize describes the size of the base mip level with format,
// as the header constructors do, so that readers relying on the header's
// pitch or linear size see the new format's size (relevant with -force).
func setPitchOrLinearSize(dds *ddsutil.Dds, format ddsutil.DxgiFormat) {
	var depth uint32
	if dds.Depth() > 1 {
		depth = dds.Depth()
	}
	const sizeFlags = ddsutil.HeaderFlagsPITCH | ddsutil.HeaderFlagsLINEARSIZE
	dds.Header.Flags &^= sizeFlags
	dds.Header.Pitch, dds.Header.LinearSize = nil, nil
	fresh, err := ddsutil.NewHeaderDxgi(ddsutil.DxgiParams{
		Width: dds.Width(), Height: dds.Height(), Depth: depth, Format: format,
	})
	if err != nil {
		// The format has no pitch (or it overflows): leave both unset.
		return
	}
	dds.Header.Flags |= fresh.Flags & sizeFlags
	dds.Header.Pitch, dds.Header.LinearSize = fresh.Pitch, fresh.LinearSize
}

// writeFileAtomic writes dds to a temporary file next to path and renames it
// over path, so a failed write never leaves a truncated file behind. If path
// is a symbolic link, the file it points to is replaced. Permission bits are
// kept, and so are the owner and group where the platform and the user's
// privileges allow (see copyOwner).
func writeFileAtomic(path string, dds *ddsutil.Dds) error {
	if target, err := filepath.EvalSymlinks(path); err == nil {
		path = target
	}
	temp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if info, err := os.Stat(path); err == nil {
		if err := temp.Chmod(info.Mode().Perm()); err != nil {
			temp.Close()
			return err
		}
		copyOwner(temp, info)
	}
	if err := dds.Write(temp); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), path)
}
