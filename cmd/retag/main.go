// Command retag re-tags the format of a DX10 DDS file that is tagged wrong.
// The data is not converted in any way.
package main

import (
	"fmt"
	"os"

	"github.com/myparsleycat/ddsutil"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: retag <ddsfile> <format>")
		os.Exit(1)
	}
	filename := os.Args[1]
	tag := os.Args[2]

	// Rather than implementing parsing for all dxgi and d3d formats, we
	// hackily just add the ones we care about here (as in the original).
	var format ddsutil.DxgiFormat
	switch tag {
	case "BC7_UNorm":
		format = ddsutil.DxgiFormatBC7_UNorm
	case "BC7_UNorm_sRGB":
		format = ddsutil.DxgiFormatBC7_UNorm_sRGB
	default:
		panic("format not implemented")
	}

	file, err := os.OpenFile(filename, os.O_RDWR, 0)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	dds, err := ddsutil.Read(file)
	if err != nil {
		panic(err)
	}

	if dds.Header10 != nil {
		dds.Header10.DxgiFormat = format
	} else {
		panic("d3d formats not implemented")
	}

	if _, err := file.Seek(0, 0); err != nil {
		panic(fmt.Sprintf("Error seeking to start of output file: %v", err))
	}

	if err := dds.Write(file); err != nil {
		panic(fmt.Sprintf("Error writing file: %v", err))
	}

	fmt.Println("Done.")
}
