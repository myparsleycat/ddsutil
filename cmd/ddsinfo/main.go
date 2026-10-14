// Command ddsinfo prints information about DDS files.
//
// Usage:
//
//	ddsinfo file.dds...
package main

import (
	"fmt"
	"os"

	"github.com/myparsleycat/ddsutil"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: ddsinfo file.dds...")
		os.Exit(2)
	}
	failed := false
	for i, filename := range os.Args[1:] {
		if i > 0 {
			fmt.Println()
		}
		if err := printInfo(filename); err != nil {
			fmt.Fprintf(os.Stderr, "ddsinfo: %s: %v\n", filename, err)
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

func printInfo(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	dds, err := ddsutil.Read(file)
	if err != nil {
		return err
	}

	if len(os.Args) > 2 {
		fmt.Printf("%s:\n", filename)
	}
	fmt.Print(dds)
	if format, err := dds.ImageFormat(); err == nil {
		fmt.Printf("  Image format: %v\n", format)
	} else {
		fmt.Printf("  Image format: not decodable (%v)\n", err)
	}
	if stride, err := dds.ArrayStride(); err == nil {
		// The DX10 array size of a cube map counts cubes, not faces.
		layers := uint64(dds.NumArrayLayers())
		if dds.Header10 != nil && dds.Header10.MiscFlag.Contains(ddsutil.MiscFlagTEXTURECUBE) {
			layers *= 6
		}
		expected := stride * layers
		fmt.Printf("  Data: %d bytes (%d expected)\n", len(dds.Data), expected)
	} else {
		fmt.Printf("  Data: %d bytes\n", len(dds.Data))
	}
	return nil
}
