// Command ddsinfo prints information about a DDS file.
package main

import (
	"fmt"
	"os"

	"github.com/myparsleycat/ddsutil"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: ddsinfo <filename>")
		os.Exit(1)
	}
	filename := os.Args[1]

	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	dds, err := ddsutil.Read(file)
	if err != nil {
		panic(err)
	}

	fmt.Print(dds)
}
