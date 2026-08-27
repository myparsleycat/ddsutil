// bcdec.go: BC1/BC2/BC3/BC4/BC5/BC6H/BC7 block decompression.
//
// Implements the algorithms of the MIT licensed "bcdec" C library by iOrange:
// https://github.com/iOrange/bcdec/blob/main/bcdec.h
//
// Used information sources:
// https://docs.microsoft.com/en-us/windows/win32/direct3d10/d3d10-graphics-programming-guide-resources-block-compression
// https://docs.microsoft.com/en-us/windows/win32/direct3d11/bc6h-format
// https://docs.microsoft.com/en-us/windows/win32/direct3d11/bc7-format
// https://docs.microsoft.com/en-us/windows/win32/direct3d11/bc7-format-mode-reference
//
// ! WARNING ! Khronos's BPTC partitions tables contain mistakes, do not use them!
// https://www.khronos.org/registry/DataFormat/specs/1.1/dataformat.1.1.html#BPTC
//
// ! Use tables from here instead !
// https://www.khronos.org/registry/OpenGL/extensions/ARB/ARB_texture_compression_bptc.txt
//
// Leaving it here as it's a nice read
// https://fgiesen.wordpress.com/2021/10/04/gpu-bcn-decoding/
//
// Fast half to float function from here
// https://gist.github.com/rygorous/2144712
package ddsutil

import (
	"encoding/binary"
	"math"
)

// BC6H: how many bits each channel actually uses per mode (14 modes).
// Rows: W, dR, dG, dB.
var bc6hActualBitsCount = [4][14]uint8{
	{10, 7, 11, 11, 11, 9, 8, 8, 8, 6, 10, 11, 12, 16}, //  W
	{5, 6, 5, 4, 4, 5, 6, 5, 5, 6, 10, 9, 8, 4},        // dR
	{5, 6, 4, 5, 4, 5, 5, 6, 5, 6, 10, 9, 8, 4},        // dG
	{5, 6, 4, 4, 5, 5, 5, 5, 6, 6, 10, 9, 8, 4},        // dB
}

// BC7: how many bits each channel actually uses per mode (8 modes).
// Rows: RGBA, Alpha.
var bc7ActualBitsCount = [2][8]uint8{
	{4, 6, 5, 7, 5, 7, 7, 5}, // RGBA
	{0, 0, 0, 0, 6, 8, 7, 5}, // Alpha
}

// There are 32 possible partition sets for a two-region tile.
// Each 4x4 block represents a single shape.
// Here also every fix-up index has MSB bit set.
var bc6hPartitionSets = [32][4][4]uint8{
	{{128, 0, 1, 1}, {0, 0, 1, 1}, {0, 0, 1, 1}, {0, 0, 1, 129}}, //  0
	{{128, 0, 0, 1}, {0, 0, 0, 1}, {0, 0, 0, 1}, {0, 0, 0, 129}}, //  1
	{{128, 1, 1, 1}, {0, 1, 1, 1}, {0, 1, 1, 1}, {0, 1, 1, 129}}, //  2
	{{128, 0, 0, 1}, {0, 0, 1, 1}, {0, 0, 1, 1}, {0, 1, 1, 129}}, //  3
	{{128, 0, 0, 0}, {0, 0, 0, 1}, {0, 0, 0, 1}, {0, 0, 1, 129}}, //  4
	{{128, 0, 1, 1}, {0, 1, 1, 1}, {0, 1, 1, 1}, {1, 1, 1, 129}}, //  5
	{{128, 0, 0, 1}, {0, 0, 1, 1}, {0, 1, 1, 1}, {1, 1, 1, 129}}, //  6
	{{128, 0, 0, 0}, {0, 0, 0, 1}, {0, 0, 1, 1}, {0, 1, 1, 129}}, //  7
	{{128, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 1}, {0, 0, 1, 129}}, //  8
	{{128, 0, 1, 1}, {0, 1, 1, 1}, {1, 1, 1, 1}, {1, 1, 1, 129}}, //  9
	{{128, 0, 0, 0}, {0, 0, 0, 1}, {0, 1, 1, 1}, {1, 1, 1, 129}}, // 10
	{{128, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 1}, {0, 1, 1, 129}}, // 11
	{{128, 0, 0, 1}, {0, 1, 1, 1}, {1, 1, 1, 1}, {1, 1, 1, 129}}, // 12
	{{128, 0, 0, 0}, {0, 0, 0, 0}, {1, 1, 1, 1}, {1, 1, 1, 129}}, // 13
	{{128, 0, 0, 0}, {1, 1, 1, 1}, {1, 1, 1, 1}, {1, 1, 1, 129}}, // 14
	{{128, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {1, 1, 1, 129}}, // 15
	{{128, 0, 0, 0}, {1, 0, 0, 0}, {1, 1, 1, 0}, {1, 1, 1, 129}}, // 16
	{{128, 1, 129, 1}, {0, 0, 0, 1}, {0, 0, 0, 0}, {0, 0, 0, 0}}, // 17
	{{128, 0, 0, 0}, {0, 0, 0, 0}, {129, 0, 0, 0}, {1, 1, 1, 0}}, // 18
	{{128, 1, 129, 1}, {0, 0, 1, 1}, {0, 0, 0, 1}, {0, 0, 0, 0}}, // 19
	{{128, 0, 129, 1}, {0, 0, 0, 1}, {0, 0, 0, 0}, {0, 0, 0, 0}}, // 20
	{{128, 0, 0, 0}, {1, 0, 0, 0}, {129, 1, 0, 0}, {1, 1, 1, 0}}, // 21
	{{128, 0, 0, 0}, {0, 0, 0, 0}, {129, 0, 0, 0}, {1, 1, 0, 0}}, // 22
	{{128, 1, 1, 1}, {0, 0, 1, 1}, {0, 0, 1, 1}, {0, 0, 0, 129}}, // 23
	{{128, 0, 129, 1}, {0, 0, 0, 1}, {0, 0, 0, 1}, {0, 0, 0, 0}}, // 24
	{{128, 0, 0, 0}, {1, 0, 0, 0}, {129, 0, 0, 0}, {1, 1, 0, 0}}, // 25
	{{128, 1, 129, 0}, {0, 1, 1, 0}, {0, 1, 1, 0}, {0, 1, 1, 0}}, // 26
	{{128, 0, 129, 1}, {0, 1, 1, 0}, {0, 1, 1, 0}, {1, 1, 0, 0}}, // 27
	{{128, 0, 0, 1}, {0, 1, 1, 1}, {129, 1, 1, 0}, {1, 0, 0, 0}}, // 28
	{{128, 0, 0, 0}, {1, 1, 1, 1}, {129, 1, 1, 1}, {0, 0, 0, 0}}, // 29
	{{128, 1, 129, 1}, {0, 0, 0, 1}, {1, 0, 0, 0}, {1, 1, 1, 0}}, // 30
	{{128, 0, 129, 1}, {1, 0, 0, 1}, {1, 0, 0, 1}, {1, 1, 0, 0}}, // 31
}

// There are 64 possible partition sets for a two-region tile.
// Each 4x4 block represents a single shape.
// Here also every fix-up index has MSB bit set.
var bc7PartitionSets = [2][64][4][4]uint8{
	{
		// Partition table for 2-subset BPTC
		{{128, 0, 1, 1}, {0, 0, 1, 1}, {0, 0, 1, 1}, {0, 0, 1, 129}}, //  0
		{{128, 0, 0, 1}, {0, 0, 0, 1}, {0, 0, 0, 1}, {0, 0, 0, 129}}, //  1
		{{128, 1, 1, 1}, {0, 1, 1, 1}, {0, 1, 1, 1}, {0, 1, 1, 129}}, //  2
		{{128, 0, 0, 1}, {0, 0, 1, 1}, {0, 0, 1, 1}, {0, 1, 1, 129}}, //  3
		{{128, 0, 0, 0}, {0, 0, 0, 1}, {0, 0, 0, 1}, {0, 0, 1, 129}}, //  4
		{{128, 0, 1, 1}, {0, 1, 1, 1}, {0, 1, 1, 1}, {1, 1, 1, 129}}, //  5
		{{128, 0, 0, 1}, {0, 0, 1, 1}, {0, 1, 1, 1}, {1, 1, 1, 129}}, //  6
		{{128, 0, 0, 0}, {0, 0, 0, 1}, {0, 0, 1, 1}, {0, 1, 1, 129}}, //  7
		{{128, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 1}, {0, 0, 1, 129}}, //  8
		{{128, 0, 1, 1}, {0, 1, 1, 1}, {1, 1, 1, 1}, {1, 1, 1, 129}}, //  9
		{{128, 0, 0, 0}, {0, 0, 0, 1}, {0, 1, 1, 1}, {1, 1, 1, 129}}, // 10
		{{128, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 1}, {0, 1, 1, 129}}, // 11
		{{128, 0, 0, 1}, {0, 1, 1, 1}, {1, 1, 1, 1}, {1, 1, 1, 129}}, // 12
		{{128, 0, 0, 0}, {0, 0, 0, 0}, {1, 1, 1, 1}, {1, 1, 1, 129}}, // 13
		{{128, 0, 0, 0}, {1, 1, 1, 1}, {1, 1, 1, 1}, {1, 1, 1, 129}}, // 14
		{{128, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {1, 1, 1, 129}}, // 15
		{{128, 0, 0, 0}, {1, 0, 0, 0}, {1, 1, 1, 0}, {1, 1, 1, 129}}, // 16
		{{128, 1, 129, 1}, {0, 0, 0, 1}, {0, 0, 0, 0}, {0, 0, 0, 0}}, // 17
		{{128, 0, 0, 0}, {0, 0, 0, 0}, {129, 0, 0, 0}, {1, 1, 1, 0}}, // 18
		{{128, 1, 129, 1}, {0, 0, 1, 1}, {0, 0, 0, 1}, {0, 0, 0, 0}}, // 19
		{{128, 0, 129, 1}, {0, 0, 0, 1}, {0, 0, 0, 0}, {0, 0, 0, 0}}, // 20
		{{128, 0, 0, 0}, {1, 0, 0, 0}, {129, 1, 0, 0}, {1, 1, 1, 0}}, // 21
		{{128, 0, 0, 0}, {0, 0, 0, 0}, {129, 0, 0, 0}, {1, 1, 0, 0}}, // 22
		{{128, 1, 1, 1}, {0, 0, 1, 1}, {0, 0, 1, 1}, {0, 0, 0, 129}}, // 23
		{{128, 0, 129, 1}, {0, 0, 0, 1}, {0, 0, 0, 1}, {0, 0, 0, 0}}, // 24
		{{128, 0, 0, 0}, {1, 0, 0, 0}, {129, 0, 0, 0}, {1, 1, 0, 0}}, // 25
		{{128, 1, 129, 0}, {0, 1, 1, 0}, {0, 1, 1, 0}, {0, 1, 1, 0}}, // 26
		{{128, 0, 129, 1}, {0, 1, 1, 0}, {0, 1, 1, 0}, {1, 1, 0, 0}}, // 27
		{{128, 0, 0, 1}, {0, 1, 1, 1}, {129, 1, 1, 0}, {1, 0, 0, 0}}, // 28
		{{128, 0, 0, 0}, {1, 1, 1, 1}, {129, 1, 1, 1}, {0, 0, 0, 0}}, // 29
		{{128, 1, 129, 1}, {0, 0, 0, 1}, {1, 0, 0, 0}, {1, 1, 1, 0}}, // 30
		{{128, 0, 129, 1}, {1, 0, 0, 1}, {1, 0, 0, 1}, {1, 1, 0, 0}}, // 31
		{{128, 1, 0, 1}, {0, 1, 0, 1}, {0, 1, 0, 1}, {0, 1, 0, 129}}, // 32
		{{128, 0, 0, 0}, {1, 1, 1, 1}, {0, 0, 0, 0}, {1, 1, 1, 129}}, // 33
		{{128, 1, 0, 1}, {1, 0, 129, 0}, {0, 1, 0, 1}, {1, 0, 1, 0}}, // 34
		{{128, 0, 1, 1}, {0, 0, 1, 1}, {129, 1, 0, 0}, {1, 1, 0, 0}}, // 35
		{{128, 0, 129, 1}, {1, 1, 0, 0}, {0, 0, 1, 1}, {1, 1, 0, 0}}, // 36
		{{128, 1, 0, 1}, {0, 1, 0, 1}, {129, 0, 1, 0}, {1, 0, 1, 0}}, // 37
		{{128, 1, 1, 0}, {1, 0, 0, 1}, {0, 1, 1, 0}, {1, 0, 0, 129}}, // 38
		{{128, 1, 0, 1}, {1, 0, 1, 0}, {1, 0, 1, 0}, {0, 1, 0, 129}}, // 39
		{{128, 1, 129, 1}, {0, 0, 1, 1}, {1, 1, 0, 0}, {1, 1, 1, 0}}, // 40
		{{128, 0, 0, 1}, {0, 0, 1, 1}, {129, 1, 0, 0}, {1, 0, 0, 0}}, // 41
		{{128, 0, 129, 1}, {0, 0, 1, 0}, {0, 1, 0, 0}, {1, 1, 0, 0}}, // 42
		{{128, 0, 129, 1}, {1, 0, 1, 1}, {1, 1, 0, 1}, {1, 1, 0, 0}}, // 43
		{{128, 1, 129, 0}, {1, 0, 0, 1}, {1, 0, 0, 1}, {0, 1, 1, 0}}, // 44
		{{128, 0, 1, 1}, {1, 1, 0, 0}, {1, 1, 0, 0}, {0, 0, 1, 129}}, // 45
		{{128, 1, 1, 0}, {0, 1, 1, 0}, {1, 0, 0, 1}, {1, 0, 0, 129}}, // 46
		{{128, 0, 0, 0}, {0, 1, 129, 0}, {0, 1, 1, 0}, {0, 0, 0, 0}}, // 47
		{{128, 1, 0, 0}, {1, 1, 129, 0}, {0, 1, 0, 0}, {0, 0, 0, 0}}, // 48
		{{128, 0, 129, 0}, {0, 1, 1, 1}, {0, 0, 1, 0}, {0, 0, 0, 0}}, // 49
		{{128, 0, 0, 0}, {0, 0, 129, 0}, {0, 1, 1, 1}, {0, 0, 1, 0}}, // 50
		{{128, 0, 0, 0}, {0, 1, 0, 0}, {129, 1, 1, 0}, {0, 1, 0, 0}}, // 51
		{{128, 1, 1, 0}, {1, 1, 0, 0}, {1, 0, 0, 1}, {0, 0, 1, 129}}, // 52
		{{128, 0, 1, 1}, {0, 1, 1, 0}, {1, 1, 0, 0}, {1, 0, 0, 129}}, // 53
		{{128, 1, 129, 0}, {0, 0, 1, 1}, {1, 0, 0, 1}, {1, 1, 0, 0}}, // 54
		{{128, 0, 129, 1}, {1, 0, 0, 1}, {1, 1, 0, 0}, {0, 1, 1, 0}}, // 55
		{{128, 1, 1, 0}, {1, 1, 0, 0}, {1, 1, 0, 0}, {1, 0, 0, 129}}, // 56
		{{128, 1, 1, 0}, {0, 0, 1, 1}, {0, 0, 1, 1}, {1, 0, 0, 129}}, // 57
		{{128, 1, 1, 1}, {1, 1, 1, 0}, {1, 0, 0, 0}, {0, 0, 0, 129}}, // 58
		{{128, 0, 0, 1}, {1, 0, 0, 0}, {1, 1, 1, 0}, {0, 1, 1, 129}}, // 59
		{{128, 0, 0, 0}, {1, 1, 1, 1}, {0, 0, 1, 1}, {0, 0, 1, 129}}, // 60
		{{128, 0, 129, 1}, {0, 0, 1, 1}, {1, 1, 1, 1}, {0, 0, 0, 0}}, // 61
		{{128, 0, 129, 0}, {0, 0, 1, 0}, {1, 1, 1, 0}, {1, 1, 1, 0}}, // 62
		{{128, 1, 0, 0}, {0, 1, 0, 0}, {0, 1, 1, 1}, {0, 1, 1, 129}}, // 63
	},
	{
		// Partition table for 3-subset BPTC
		{{128, 0, 1, 129}, {0, 0, 1, 1}, {0, 2, 2, 1}, {2, 2, 2, 130}}, //  0
		{{128, 0, 0, 129}, {0, 0, 1, 1}, {130, 2, 1, 1}, {2, 2, 2, 1}}, //  1
		{{128, 0, 0, 0}, {2, 0, 0, 1}, {130, 2, 1, 1}, {2, 2, 1, 129}}, //  2
		{{128, 2, 2, 130}, {0, 0, 2, 2}, {0, 0, 1, 1}, {0, 1, 1, 129}}, //  3
		{{128, 0, 0, 0}, {0, 0, 0, 0}, {129, 1, 2, 2}, {1, 1, 2, 130}}, //  4
		{{128, 0, 1, 129}, {0, 0, 1, 1}, {0, 0, 2, 2}, {0, 0, 2, 130}}, //  5
		{{128, 0, 2, 130}, {0, 0, 2, 2}, {1, 1, 1, 1}, {1, 1, 1, 129}}, //  6
		{{128, 0, 1, 1}, {0, 0, 1, 1}, {130, 2, 1, 1}, {2, 2, 1, 129}}, //  7
		{{128, 0, 0, 0}, {0, 0, 0, 0}, {129, 1, 1, 1}, {2, 2, 2, 130}}, //  8
		{{128, 0, 0, 0}, {1, 1, 1, 1}, {129, 1, 1, 1}, {2, 2, 2, 130}}, //  9
		{{128, 0, 0, 0}, {1, 1, 129, 1}, {2, 2, 2, 2}, {2, 2, 2, 130}}, // 10
		{{128, 0, 1, 2}, {0, 0, 129, 2}, {0, 0, 1, 2}, {0, 0, 1, 130}}, // 11
		{{128, 1, 1, 2}, {0, 1, 129, 2}, {0, 1, 1, 2}, {0, 1, 1, 130}}, // 12
		{{128, 1, 2, 2}, {0, 129, 2, 2}, {0, 1, 2, 2}, {0, 1, 2, 130}}, // 13
		{{128, 0, 1, 129}, {0, 1, 1, 2}, {1, 1, 2, 2}, {1, 2, 2, 130}}, // 14
		{{128, 0, 1, 129}, {2, 0, 0, 1}, {130, 2, 0, 0}, {2, 2, 2, 0}}, // 15
		{{128, 0, 0, 129}, {0, 0, 1, 1}, {0, 1, 1, 2}, {1, 1, 2, 130}}, // 16
		{{128, 1, 1, 129}, {0, 0, 1, 1}, {130, 0, 0, 1}, {2, 2, 0, 0}}, // 17
		{{128, 0, 0, 0}, {1, 1, 2, 2}, {129, 1, 2, 2}, {1, 1, 2, 130}}, // 18
		{{128, 0, 2, 130}, {0, 0, 2, 2}, {0, 0, 2, 2}, {1, 1, 1, 129}}, // 19
		{{128, 1, 1, 129}, {0, 1, 1, 1}, {0, 2, 2, 2}, {0, 2, 2, 130}}, // 20
		{{128, 0, 0, 129}, {0, 0, 0, 1}, {130, 2, 2, 1}, {2, 2, 2, 1}}, // 21
		{{128, 0, 0, 0}, {0, 0, 129, 1}, {0, 1, 2, 2}, {0, 1, 2, 130}}, // 22
		{{128, 0, 0, 0}, {1, 1, 0, 0}, {130, 2, 129, 0}, {2, 2, 1, 0}}, // 23
		{{128, 1, 2, 130}, {0, 129, 2, 2}, {0, 0, 1, 1}, {0, 0, 0, 0}}, // 24
		{{128, 0, 1, 2}, {0, 0, 1, 2}, {129, 1, 2, 2}, {2, 2, 2, 130}}, // 25
		{{128, 1, 1, 0}, {1, 2, 130, 1}, {129, 2, 2, 1}, {0, 1, 1, 0}}, // 26
		{{128, 0, 0, 0}, {0, 1, 129, 0}, {1, 2, 130, 1}, {1, 2, 2, 1}}, // 27
		{{128, 0, 2, 2}, {1, 1, 0, 2}, {129, 1, 0, 2}, {0, 0, 2, 130}}, // 28
		{{128, 1, 1, 0}, {0, 129, 1, 0}, {2, 0, 0, 2}, {2, 2, 2, 130}}, // 29
		{{128, 0, 1, 1}, {0, 1, 2, 2}, {0, 1, 130, 2}, {0, 0, 1, 129}}, // 30
		{{128, 0, 0, 0}, {2, 0, 0, 0}, {130, 2, 1, 1}, {2, 2, 2, 129}}, // 31
		{{128, 0, 0, 0}, {0, 0, 0, 2}, {129, 1, 2, 2}, {1, 2, 2, 130}}, // 32
		{{128, 2, 2, 130}, {0, 0, 2, 2}, {0, 0, 1, 2}, {0, 0, 1, 129}}, // 33
		{{128, 0, 1, 129}, {0, 0, 1, 2}, {0, 0, 2, 2}, {0, 2, 2, 130}}, // 34
		{{128, 1, 2, 0}, {0, 129, 2, 0}, {0, 1, 130, 0}, {0, 1, 2, 0}}, // 35
		{{128, 0, 0, 0}, {1, 1, 129, 1}, {2, 2, 130, 2}, {0, 0, 0, 0}}, // 36
		{{128, 1, 2, 0}, {1, 2, 0, 1}, {130, 0, 129, 2}, {0, 1, 2, 0}}, // 37
		{{128, 1, 2, 0}, {2, 0, 1, 2}, {129, 130, 0, 1}, {0, 1, 2, 0}}, // 38
		{{128, 0, 1, 1}, {2, 2, 0, 0}, {1, 1, 130, 2}, {0, 0, 1, 129}}, // 39
		{{128, 0, 1, 1}, {1, 1, 130, 2}, {2, 2, 0, 0}, {0, 0, 1, 129}}, // 40
		{{128, 1, 0, 129}, {0, 1, 0, 1}, {2, 2, 2, 2}, {2, 2, 2, 130}}, // 41
		{{128, 0, 0, 0}, {0, 0, 0, 0}, {130, 1, 2, 1}, {2, 1, 2, 129}}, // 42
		{{128, 0, 2, 2}, {1, 129, 2, 2}, {0, 0, 2, 2}, {1, 1, 2, 130}}, // 43
		{{128, 0, 2, 130}, {0, 0, 1, 1}, {0, 0, 2, 2}, {0, 0, 1, 129}}, // 44
		{{128, 2, 2, 0}, {1, 2, 130, 1}, {0, 2, 2, 0}, {1, 2, 2, 129}}, // 45
		{{128, 1, 0, 1}, {2, 2, 130, 2}, {2, 2, 2, 2}, {0, 1, 0, 129}}, // 46
		{{128, 0, 0, 0}, {2, 1, 2, 1}, {130, 1, 2, 1}, {2, 1, 2, 129}}, // 47
		{{128, 1, 0, 129}, {0, 1, 0, 1}, {0, 1, 0, 1}, {2, 2, 2, 130}}, // 48
		{{128, 2, 2, 130}, {0, 1, 1, 1}, {0, 2, 2, 2}, {0, 1, 1, 129}}, // 49
		{{128, 0, 0, 2}, {1, 129, 1, 2}, {0, 0, 0, 2}, {1, 1, 1, 130}}, // 50
		{{128, 0, 0, 0}, {2, 129, 1, 2}, {2, 1, 1, 2}, {2, 1, 1, 130}}, // 51
		{{128, 2, 2, 2}, {0, 129, 1, 1}, {0, 1, 1, 1}, {0, 2, 2, 130}}, // 52
		{{128, 0, 0, 2}, {1, 1, 1, 2}, {129, 1, 1, 2}, {0, 0, 0, 130}}, // 53
		{{128, 1, 1, 0}, {0, 129, 1, 0}, {0, 1, 1, 0}, {2, 2, 2, 130}}, // 54
		{{128, 0, 0, 0}, {0, 0, 0, 0}, {2, 1, 129, 2}, {2, 1, 1, 130}}, // 55
		{{128, 1, 1, 0}, {0, 129, 1, 0}, {2, 2, 2, 2}, {2, 2, 2, 130}}, // 56
		{{128, 0, 2, 2}, {0, 0, 1, 1}, {0, 0, 129, 1}, {0, 0, 2, 130}}, // 57
		{{128, 0, 2, 2}, {1, 1, 2, 2}, {129, 1, 2, 2}, {0, 0, 2, 130}}, // 58
		{{128, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {2, 129, 1, 130}}, // 59
		{{128, 0, 0, 130}, {0, 0, 0, 1}, {0, 0, 0, 2}, {0, 0, 0, 129}}, // 60
		{{128, 2, 2, 2}, {1, 2, 2, 2}, {0, 2, 2, 2}, {129, 2, 2, 130}}, // 61
		{{128, 1, 0, 129}, {2, 2, 2, 2}, {2, 2, 2, 2}, {2, 2, 2, 130}}, // 62
		{{128, 1, 1, 129}, {2, 0, 1, 1}, {130, 2, 0, 1}, {2, 2, 2, 0}}, // 63
	},
}

// BC6H interpolation weights.
var bc6hAWeight3 = [8]int32{0, 9, 18, 27, 37, 46, 55, 64}
var bc6hAWeight4 = [16]int32{0, 4, 9, 13, 17, 21, 26, 30, 34, 38, 43, 47, 51, 55, 60, 64}

// BC7 interpolation weights.
var bc7AWeight2 = [4]uint32{0, 21, 43, 64}
var bc7AWeight3 = [8]uint32{0, 9, 18, 27, 37, 46, 55, 64}
var bc7AWeight4 = [16]uint32{0, 4, 9, 13, 17, 21, 26, 30, 34, 38, 43, 47, 51, 55, 60, 64}

// BC4 interpolation weights.
var bc4AWeights4 = [4]int32{13107, 26215, 39321, 52429}
var bc4AWeights6 = [6]int32{9363, 18724, 28086, 37450, 46812, 56173}

// BC7 modes that have P-bits.
const bc7SModeHasPbits = 0b11001011

// bc1 decodes 8 bytes from compressedBlock to RGBA8
// with destinationPitch many bytes per output row.
func bc1(compressedBlock []byte, decompressedBlock []byte, destinationPitch int) {
	colorBlock(compressedBlock, decompressedBlock, destinationPitch, false)
}

// bc2 decodes 16 bytes from compressedBlock to RGBA8
// with destinationPitch many bytes per output row.
func bc2(compressedBlock []byte, decompressedBlock []byte, destinationPitch int) {
	colorBlock(compressedBlock[8:], decompressedBlock, destinationPitch, true)
	sharpAlphaBlock(compressedBlock, decompressedBlock, destinationPitch)
}

// bc3 decodes 16 bytes from compressedBlock to RGBA8
// with destinationPitch many bytes per output row.
func bc3(compressedBlock []byte, decompressedBlock []byte, destinationPitch int) {
	colorBlock(compressedBlock[8:], decompressedBlock, destinationPitch, true)
	smoothAlphaBlock(compressedBlock, decompressedBlock[3:], destinationPitch, 4)
}

// bc4 decodes 8 bytes from compressedBlock to R8
// with destinationPitch many bytes per output row.
func bc4(compressedBlock []byte, decompressedBlock []byte, destinationPitch int, isSigned bool) {
	bc4Block(compressedBlock, decompressedBlock, destinationPitch, 1, isSigned)
}

// bc4Float decodes 8 bytes from compressedBlock to R Float32
// with destinationPitch many floats per output row.
func bc4Float(compressedBlock []byte, decompressedBlock []float32, destinationPitch int, isSigned bool) {
	bc4BlockFloat(compressedBlock, decompressedBlock, destinationPitch, 1, isSigned)
}

// bc5 decodes 16 bytes from compressedBlock to RG8
// with destinationPitch many bytes per output row.
func bc5(compressedBlock []byte, decompressedBlock []byte, destinationPitch int, isSigned bool) {
	bc4Block(compressedBlock, decompressedBlock, destinationPitch, 2, isSigned)
	bc4Block(compressedBlock[8:], decompressedBlock[1:], destinationPitch, 2, isSigned)
}

// bc5Float decodes 16 bytes from compressedBlock to RG Float32
// with destinationPitch many floats per output row.
func bc5Float(compressedBlock []byte, decompressedBlock []float32, destinationPitch int, isSigned bool) {
	bc4BlockFloat(compressedBlock, decompressedBlock, destinationPitch, 2, isSigned)
	bc4BlockFloat(compressedBlock[8:], decompressedBlock[1:], destinationPitch, 2, isSigned)
}

// bc6hHalf decodes 16 bytes from compressedBlock to RGBFloat16
// with destinationPitch many half floats per output row.
//
// The uint16 values contain the bits of half-precision floats.
func bc6hHalf(compressedBlock []byte, decompressedBlock []uint16, destinationPitch int, isSigned bool) {
	bstream := bitstream{
		low:  binary.LittleEndian.Uint64(compressedBlock[0:8]),
		high: binary.LittleEndian.Uint64(compressedBlock[8:16]),
	}

	var r [4]int32 // wxyz
	var g [4]int32 // wxyz
	var b [4]int32 // wxyz

	mode := bstream.readBits(2)
	if mode > 1 {
		mode |= bstream.readBits(3) << 2
	}

	// modes >= 11 (10 in my code) are using 0 one, others will read it from the bitstream
	partition := int32(0)

	switch mode {
	// mode 1
	case 0b00:
		// Partitition indices: 46 bits
		// Partition: 5 bits
		// Color Endpoints: 75 bits (10.555, 10.555, 10.555)
		g[2] |= bstream.readBitI32() << 4  // gy[4]
		b[2] |= bstream.readBitI32() << 4  // by[4]
		b[3] |= bstream.readBitI32() << 4  // bz[4]
		r[0] |= bstream.readBitsI32(10)    // rw[9:0]
		g[0] |= bstream.readBitsI32(10)    // gw[9:0]
		b[0] |= bstream.readBitsI32(10)    // bw[9:0]
		r[1] |= bstream.readBitsI32(5)     // rx[4:0]
		g[3] |= bstream.readBitI32() << 4  // gz[4]
		g[2] |= bstream.readBitsI32(4)     // gy[3:0]
		g[1] |= bstream.readBitsI32(5)     // gx[4:0]
		b[3] |= bstream.readBitI32()       // bz[0]
		g[3] |= bstream.readBitsI32(4)     // gz[3:0]
		b[1] |= bstream.readBitsI32(5)     // bx[4:0]
		b[3] |= bstream.readBitI32() << 1  // bz[1]
		b[2] |= bstream.readBitsI32(4)     // by[3:0]
		r[2] |= bstream.readBitsI32(5)     // ry[4:0]
		b[3] |= bstream.readBitI32() << 2  // bz[2]
		r[3] |= bstream.readBitsI32(5)     // rz[4:0]
		b[3] |= bstream.readBitI32() << 3  // bz[3]
		partition = bstream.readBitsI32(5) // d[4:0]
		mode = 0

	// mode 2
	case 0b01:
		// Partitition indices: 46 bits
		// Partition: 5 bits
		// Color Endpoints: 75 bits (7666, 7666, 7666)
		g[2] |= bstream.readBitI32() << 5  // gy[5]
		g[3] |= bstream.readBitI32() << 4  // gz[4]
		g[3] |= bstream.readBitI32() << 5  // gz[5]
		r[0] |= bstream.readBitsI32(7)     // rw[6:0]
		b[3] |= bstream.readBitI32()       // bz[0]
		b[3] |= bstream.readBitI32() << 1  // bz[1]
		b[2] |= bstream.readBitI32() << 4  // by[4]
		g[0] |= bstream.readBitsI32(7)     // gw[6:0]
		b[2] |= bstream.readBitI32() << 5  // by[5]
		b[3] |= bstream.readBitI32() << 2  // bz[2]
		g[2] |= bstream.readBitI32() << 4  // gy[4]
		b[0] |= bstream.readBitsI32(7)     // bw[6:0]
		b[3] |= bstream.readBitI32() << 3  // bz[3]
		b[3] |= bstream.readBitI32() << 5  // bz[5]
		b[3] |= bstream.readBitI32() << 4  // bz[4]
		r[1] |= bstream.readBitsI32(6)     // rx[5:0]
		g[2] |= bstream.readBitsI32(4)     // gy[3:0]
		g[1] |= bstream.readBitsI32(6)     // gx[5:0]
		g[3] |= bstream.readBitsI32(4)     // gz[3:0]
		b[1] |= bstream.readBitsI32(6)     // bx[5:0]
		b[2] |= bstream.readBitsI32(4)     // by[3:0]
		r[2] |= bstream.readBitsI32(6)     // ry[5:0]
		r[3] |= bstream.readBitsI32(6)     // rz[5:0]
		partition = bstream.readBitsI32(5) // d[4:0]
		mode = 1

	// mode 3
	case 0b00010:
		// Partitition indices: 46 bits
		// Partition: 5 bits
		// Color Endpoints: 72 bits (11.555, 11.444, 11.444)
		r[0] |= bstream.readBitsI32(10)    // rw[9:0]
		g[0] |= bstream.readBitsI32(10)    // gw[9:0]
		b[0] |= bstream.readBitsI32(10)    // bw[9:0]
		r[1] |= bstream.readBitsI32(5)     // rx[4:0]
		r[0] |= bstream.readBitI32() << 10 // rw[10]
		g[2] |= bstream.readBitsI32(4)     // gy[3:0]
		g[1] |= bstream.readBitsI32(4)     // gx[3:0]
		g[0] |= bstream.readBitI32() << 10 // gw[10]
		b[3] |= bstream.readBitI32()       // bz[0]
		g[3] |= bstream.readBitsI32(4)     // gz[3:0]
		b[1] |= bstream.readBitsI32(4)     // bx[3:0]
		b[0] |= bstream.readBitI32() << 10 // bw[10]
		b[3] |= bstream.readBitI32() << 1  // bz[1]
		b[2] |= bstream.readBitsI32(4)     // by[3:0]
		r[2] |= bstream.readBitsI32(5)     // ry[4:0]
		b[3] |= bstream.readBitI32() << 2  // bz[2]
		r[3] |= bstream.readBitsI32(5)     // rz[4:0]
		b[3] |= bstream.readBitI32() << 3  // bz[3]
		partition = bstream.readBitsI32(5) // d[4:0]
		mode = 2

	// mode 4
	case 0b00110:
		// Partitition indices: 46 bits
		// Partition: 5 bits
		// Color Endpoints: 72 bits (11.444, 11.555, 11.444)
		r[0] |= bstream.readBitsI32(10)    // rw[9:0]
		g[0] |= bstream.readBitsI32(10)    // gw[9:0]
		b[0] |= bstream.readBitsI32(10)    // bw[9:0]
		r[1] |= bstream.readBitsI32(4)     // rx[3:0]
		r[0] |= bstream.readBitI32() << 10 // rw[10]
		g[3] |= bstream.readBitI32() << 4  // gz[4]
		g[2] |= bstream.readBitsI32(4)     // gy[3:0]
		g[1] |= bstream.readBitsI32(5)     // gx[4:0]
		g[0] |= bstream.readBitI32() << 10 // gw[10]
		g[3] |= bstream.readBitsI32(4)     // gz[3:0]
		b[1] |= bstream.readBitsI32(4)     // bx[3:0]
		b[0] |= bstream.readBitI32() << 10 // bw[10]
		b[3] |= bstream.readBitI32() << 1  // bz[1]
		b[2] |= bstream.readBitsI32(4)     // by[3:0]
		r[2] |= bstream.readBitsI32(4)     // ry[3:0]
		b[3] |= bstream.readBitI32()       // bz[0]
		b[3] |= bstream.readBitI32() << 2  // bz[2]
		r[3] |= bstream.readBitsI32(4)     // rz[3:0]
		g[2] |= bstream.readBitI32() << 4  // gy[4]
		b[3] |= bstream.readBitI32() << 3  // bz[3]
		partition = bstream.readBitsI32(5) // d[4:0]
		mode = 3

	// mode 5
	case 0b01010:
		// Partitition indices: 46 bits
		// Partition: 5 bits
		// Color Endpoints: 72 bits (11.444, 11.444, 11.555)
		r[0] |= bstream.readBitsI32(10)    // rw[9:0]
		g[0] |= bstream.readBitsI32(10)    // gw[9:0]
		b[0] |= bstream.readBitsI32(10)    // bw[9:0]
		r[1] |= bstream.readBitsI32(4)     // rx[3:0]
		r[0] |= bstream.readBitI32() << 10 // rw[10]
		b[2] |= bstream.readBitI32() << 4  // by[4]
		g[2] |= bstream.readBitsI32(4)     // gy[3:0]
		g[1] |= bstream.readBitsI32(4)     // gx[3:0]
		g[0] |= bstream.readBitI32() << 10 // gw[10]
		b[3] |= bstream.readBitI32()       // bz[0]
		g[3] |= bstream.readBitsI32(4)     // gz[3:0]
		b[1] |= bstream.readBitsI32(5)     // bx[4:0]
		b[0] |= bstream.readBitI32() << 10 // bw[10]
		b[2] |= bstream.readBitsI32(4)     // by[3:0]
		r[2] |= bstream.readBitsI32(4)     // ry[3:0]
		b[3] |= bstream.readBitI32() << 1  // bz[1]
		b[3] |= bstream.readBitI32() << 2  // bz[2]
		r[3] |= bstream.readBitsI32(4)     // rz[3:0]
		b[3] |= bstream.readBitI32() << 4  // bz[4]
		b[3] |= bstream.readBitI32() << 3  // bz[3]
		partition = bstream.readBitsI32(5) // d[4:0]
		mode = 4

	// mode 6
	case 0b01110:
		// Partitition indices: 46 bits
		// Partition: 5 bits
		// Color Endpoints: 72 bits (9555, 9555, 9555)
		r[0] |= bstream.readBitsI32(9)     // rw[8:0]
		b[2] |= bstream.readBitI32() << 4  // by[4]
		g[0] |= bstream.readBitsI32(9)     // gw[8:0]
		g[2] |= bstream.readBitI32() << 4  // gy[4]
		b[0] |= bstream.readBitsI32(9)     // bw[8:0]
		b[3] |= bstream.readBitI32() << 4  // bz[4]
		r[1] |= bstream.readBitsI32(5)     // rx[4:0]
		g[3] |= bstream.readBitI32() << 4  // gz[4]
		g[2] |= bstream.readBitsI32(4)     // gy[3:0]
		g[1] |= bstream.readBitsI32(5)     // gx[4:0]
		b[3] |= bstream.readBitI32()       // bz[0]
		g[3] |= bstream.readBitsI32(4)     // gx[3:0]
		b[1] |= bstream.readBitsI32(5)     // bx[4:0]
		b[3] |= bstream.readBitI32() << 1  // bz[1]
		b[2] |= bstream.readBitsI32(4)     // by[3:0]
		r[2] |= bstream.readBitsI32(5)     // ry[4:0]
		b[3] |= bstream.readBitI32() << 2  // bz[2]
		r[3] |= bstream.readBitsI32(5)     // rz[4:0]
		b[3] |= bstream.readBitI32() << 3  // bz[3]
		partition = bstream.readBitsI32(5) // d[4:0]
		mode = 5

	// mode 7
	case 0b10010:
		// Partitition indices: 46 bits
		// Partition: 5 bits
		// Color Endpoints: 72 bits (8666, 8555, 8555)
		r[0] |= bstream.readBitsI32(8)     // rw[7:0]
		g[3] |= bstream.readBitI32() << 4  // gz[4]
		b[2] |= bstream.readBitI32() << 4  // by[4]
		g[0] |= bstream.readBitsI32(8)     // gw[7:0]
		b[3] |= bstream.readBitI32() << 2  // bz[2]
		g[2] |= bstream.readBitI32() << 4  // gy[4]
		b[0] |= bstream.readBitsI32(8)     // bw[7:0]
		b[3] |= bstream.readBitI32() << 3  // bz[3]
		b[3] |= bstream.readBitI32() << 4  // bz[4]
		r[1] |= bstream.readBitsI32(6)     // rx[5:0]
		g[2] |= bstream.readBitsI32(4)     // gy[3:0]
		g[1] |= bstream.readBitsI32(5)     // gx[4:0]
		b[3] |= bstream.readBitI32()       // bz[0]
		g[3] |= bstream.readBitsI32(4)     // gz[3:0]
		b[1] |= bstream.readBitsI32(5)     // bx[4:0]
		b[3] |= bstream.readBitI32() << 1  // bz[1]
		b[2] |= bstream.readBitsI32(4)     // by[3:0]
		r[2] |= bstream.readBitsI32(6)     // ry[5:0]
		r[3] |= bstream.readBitsI32(6)     // rz[5:0]
		partition = bstream.readBitsI32(5) // d[4:0]
		mode = 6

	// mode 8
	case 0b10110:
		// Partitition indices: 46 bits
		// Partition: 5 bits
		// Color Endpoints: 72 bits (8555, 8666, 8555)
		r[0] |= bstream.readBitsI32(8)     // rw[7:0]
		b[3] |= bstream.readBitI32()       // bz[0]
		b[2] |= bstream.readBitI32() << 4  // by[4]
		g[0] |= bstream.readBitsI32(8)     // gw[7:0]
		g[2] |= bstream.readBitI32() << 5  // gy[5]
		g[2] |= bstream.readBitI32() << 4  // gy[4]
		b[0] |= bstream.readBitsI32(8)     // bw[7:0]
		g[3] |= bstream.readBitI32() << 5  // gz[5]
		b[3] |= bstream.readBitI32() << 4  // bz[4]
		r[1] |= bstream.readBitsI32(5)     // rx[4:0]
		g[3] |= bstream.readBitI32() << 4  // gz[4]
		g[2] |= bstream.readBitsI32(4)     // gy[3:0]
		g[1] |= bstream.readBitsI32(6)     // gx[5:0]
		g[3] |= bstream.readBitsI32(4)     // zx[3:0]
		b[1] |= bstream.readBitsI32(5)     // bx[4:0]
		b[3] |= bstream.readBitI32() << 1  // bz[1]
		b[2] |= bstream.readBitsI32(4)     // by[3:0]
		r[2] |= bstream.readBitsI32(5)     // ry[4:0]
		b[3] |= bstream.readBitI32() << 2  // bz[2]
		r[3] |= bstream.readBitsI32(5)     // rz[4:0]
		b[3] |= bstream.readBitI32() << 3  // bz[3]
		partition = bstream.readBitsI32(5) // d[4:0]
		mode = 7

	// mode 9
	case 0b11010:
		// Partitition indices: 46 bits
		// Partition: 5 bits
		// Color Endpoints: 72 bits (8555, 8555, 8666)
		r[0] |= bstream.readBitsI32(8)     // rw[7:0]
		b[3] |= bstream.readBitI32() << 1  // bz[1]
		b[2] |= bstream.readBitI32() << 4  // by[4]
		g[0] |= bstream.readBitsI32(8)     // gw[7:0]
		b[2] |= bstream.readBitI32() << 5  // by[5]
		g[2] |= bstream.readBitI32() << 4  // gy[4]
		b[0] |= bstream.readBitsI32(8)     // bw[7:0]
		b[3] |= bstream.readBitI32() << 5  // bz[5]
		b[3] |= bstream.readBitI32() << 4  // bz[4]
		r[1] |= bstream.readBitsI32(5)     // bw[4:0]
		g[3] |= bstream.readBitI32() << 4  // gz[4]
		g[2] |= bstream.readBitsI32(4)     // gy[3:0]
		g[1] |= bstream.readBitsI32(5)     // gx[4:0]
		b[3] |= bstream.readBitI32()       // bz[0]
		g[3] |= bstream.readBitsI32(4)     // gz[3:0]
		b[1] |= bstream.readBitsI32(6)     // bx[5:0]
		b[2] |= bstream.readBitsI32(4)     // by[3:0]
		r[2] |= bstream.readBitsI32(5)     // ry[4:0]
		b[3] |= bstream.readBitI32() << 2  // bz[2]
		r[3] |= bstream.readBitsI32(5)     // rz[4:0]
		b[3] |= bstream.readBitI32() << 3  // bz[3]
		partition = bstream.readBitsI32(5) // d[4:0]
		mode = 8

	// mode 10
	case 0b11110:
		// Partitition indices: 46 bits
		// Partition: 5 bits
		// Color Endpoints: 72 bits (6666, 6666, 6666)
		r[0] |= bstream.readBitsI32(6)     // rw[5:0]
		g[3] |= bstream.readBitI32() << 4  // gz[4]
		b[3] |= bstream.readBitI32()       // bz[0]
		b[3] |= bstream.readBitI32() << 1  // bz[1]
		b[2] |= bstream.readBitI32() << 4  // by[4]
		g[0] |= bstream.readBitsI32(6)     // gw[5:0]
		g[2] |= bstream.readBitI32() << 5  // gy[5]
		b[2] |= bstream.readBitI32() << 5  // by[5]
		b[3] |= bstream.readBitI32() << 2  // bz[2]
		g[2] |= bstream.readBitI32() << 4  // gy[4]
		b[0] |= bstream.readBitsI32(6)     // bw[5:0]
		g[3] |= bstream.readBitI32() << 5  // gz[5]
		b[3] |= bstream.readBitI32() << 3  // bz[3]
		b[3] |= bstream.readBitI32() << 5  // bz[5]
		b[3] |= bstream.readBitI32() << 4  // bz[4]
		r[1] |= bstream.readBitsI32(6)     // rx[5:0]
		g[2] |= bstream.readBitsI32(4)     // gy[3:0]
		g[1] |= bstream.readBitsI32(6)     // gx[5:0]
		g[3] |= bstream.readBitsI32(4)     // gz[3:0]
		b[1] |= bstream.readBitsI32(6)     // bx[5:0]
		b[2] |= bstream.readBitsI32(4)     // by[3:0]
		r[2] |= bstream.readBitsI32(6)     // ry[5:0]
		r[3] |= bstream.readBitsI32(6)     // rz[5:0]
		partition = bstream.readBitsI32(5) // d[4:0]
		mode = 9

	// mode 11
	case 0b00011:
		// Partitition indices: 63 bits
		// Partition: 0 bits
		// Color Endpoints: 60 bits (10.10, 10.10, 10.10)
		r[0] |= bstream.readBitsI32(10) // rw[9:0]
		g[0] |= bstream.readBitsI32(10) // gw[9:0]
		b[0] |= bstream.readBitsI32(10) // bw[9:0]
		r[1] |= bstream.readBitsI32(10) // rx[9:0]
		g[1] |= bstream.readBitsI32(10) // gx[9:0]
		b[1] |= bstream.readBitsI32(10) // bx[9:0]
		mode = 10

	// mode 12
	case 0b00111:
		// Partitition indices: 63 bits
		// Partition: 0 bits
		// Color Endpoints: 60 bits (11.9, 11.9, 11.9)
		r[0] |= bstream.readBitsI32(10)    // rw[9:0]
		g[0] |= bstream.readBitsI32(10)    // gw[9:0]
		b[0] |= bstream.readBitsI32(10)    // bw[9:0]
		r[1] |= bstream.readBitsI32(9)     // rx[8:0]
		r[0] |= bstream.readBitI32() << 10 // rw[10]
		g[1] |= bstream.readBitsI32(9)     // gx[8:0]
		g[0] |= bstream.readBitI32() << 10 // gw[10]
		b[1] |= bstream.readBitsI32(9)     // bx[8:0]
		b[0] |= bstream.readBitI32() << 10 // bw[10]
		mode = 11

	// mode 13
	case 0b01011:
		// Partitition indices: 63 bits
		// Partition: 0 bits
		// Color Endpoints: 60 bits (12.8, 12.8, 12.8)
		r[0] |= bstream.readBitsI32(10)    // rw[9:0]
		g[0] |= bstream.readBitsI32(10)    // gw[9:0]
		b[0] |= bstream.readBitsI32(10)    // bw[9:0]
		r[1] |= bstream.readBitsI32(8)     // rx[7:0]
		r[0] |= bstream.readBitsR(2) << 10 // rx[10:11]
		g[1] |= bstream.readBitsI32(8)     // gx[7:0]
		g[0] |= bstream.readBitsR(2) << 10 // gx[10:11]
		b[1] |= bstream.readBitsI32(8)     // bx[7:0]
		b[0] |= bstream.readBitsR(2) << 10 // bx[10:11]
		mode = 12

	// mode 14
	case 0b01111:
		// Partitition indices: 63 bits
		// Partition: 0 bits
		// Color Endpoints: 60 bits (16.4, 16.4, 16.4)
		r[0] |= bstream.readBitsI32(10)    // rw[9:0]
		g[0] |= bstream.readBitsI32(10)    // gw[9:0]
		b[0] |= bstream.readBitsI32(10)    // bw[9:0]
		r[1] |= bstream.readBitsI32(4)     // rx[3:0]
		r[0] |= bstream.readBitsR(6) << 10 // rw[10:15]
		g[1] |= bstream.readBitsI32(4)     // gx[3:0]
		g[0] |= bstream.readBitsR(6) << 10 // gw[10:15]
		b[1] |= bstream.readBitsI32(4)     // bx[3:0]
		b[0] |= bstream.readBitsR(6) << 10 // bw[10:15]
		mode = 13

	default:
		// Modes 10011, 10111, 11011, and 11111 (not shown) are reserved.
		// Do not use these in your encoder. If the hardware is passed blocks
		// with one of these modes specified, the resulting decompressed block
		// must contain all zeroes in all channels except for the alpha channel.
		for i := 0; i < 4; i++ {
			index := i * destinationPitch
			for k := 0; k < 4*3; k++ {
				decompressedBlock[index+k] = 0
			}
		}

		return
	}

	numPartitions := 0
	if mode >= 10 {
		numPartitions = 0
	} else {
		numPartitions = 1
	}

	actualBits0Mode := int32(bc6hActualBitsCount[0][mode])
	if isSigned {
		r[0] = extendSign(r[0], actualBits0Mode)
		g[0] = extendSign(g[0], actualBits0Mode)
		b[0] = extendSign(b[0], actualBits0Mode)
	}
	// Mode 11 (like Mode 10) does not use delta compression,
	// and instead stores both color endpoints explicitly.
	if (mode != 9 && mode != 10) || isSigned {
		for i := 1; i < (numPartitions+1)*2; i++ {
			r[i] = extendSign(r[i], int32(bc6hActualBitsCount[1][mode]))
			g[i] = extendSign(g[i], int32(bc6hActualBitsCount[2][mode]))
			b[i] = extendSign(b[i], int32(bc6hActualBitsCount[3][mode]))
		}
	}

	if mode != 9 && mode != 10 {
		for i := 1; i < (numPartitions+1)*2; i++ {
			r[i] = transformInverse(r[i], r[0], actualBits0Mode, isSigned)
			g[i] = transformInverse(g[i], g[0], actualBits0Mode, isSigned)
			b[i] = transformInverse(b[i], b[0], actualBits0Mode, isSigned)
		}
	}

	for i := 0; i < (numPartitions+1)*2; i++ {
		r[i] = unquantize(r[i], actualBits0Mode, isSigned)
		g[i] = unquantize(g[i], actualBits0Mode, isSigned)
		b[i] = unquantize(b[i], actualBits0Mode, isSigned)
	}

	var weights []int32
	if mode >= 10 {
		weights = bc6hAWeight4[:]
	} else {
		weights = bc6hAWeight3[:]
	}
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			partitionSet := 0
			if mode >= 10 {
				if (i | j) != 0 {
					partitionSet = 0
				} else {
					partitionSet = 128
				}
			} else {
				partitionSet = int(bc6hPartitionSets[partition][i][j])
			}

			indexBits := uint32(3)
			if mode >= 10 {
				indexBits = 4
			}
			// fix-up index is specified with one less bit
			// The fix-up index for subset 0 is always index 0
			if partitionSet&0x80 != 0 {
				indexBits--
			}
			partitionSet &= 0x01

			index := bstream.readBits(indexBits)
			weight := weights[index]

			epI := partitionSet * 2

			out := i*destinationPitch + j*3
			decompressedBlock[out] = finishUnquantize(interpolateI32(r[epI], r[epI+1], weight), isSigned)
			decompressedBlock[out+1] = finishUnquantize(interpolateI32(g[epI], g[epI+1], weight), isSigned)
			decompressedBlock[out+2] = finishUnquantize(interpolateI32(b[epI], b[epI+1], weight), isSigned)
		}
	}
}

// bc6hFloat decodes 16 bytes from compressedBlock to RGB Float32
// with destinationPitch many floats per output row.
func bc6hFloat(compressedBlock []byte, decompressedBlock []float32, destinationPitch int, isSigned bool) {
	inputPitch := 4 * 3
	var block [4 * 4 * 3]uint16
	bc6hHalf(compressedBlock, block[:], inputPitch, isSigned)

	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			// The input is f16 but the output is f32.
			inIndex := i*inputPitch + j*3
			outIndex := i*destinationPitch + j*3
			decompressedBlock[outIndex] = halfToFloatQuick(block[inIndex])
			decompressedBlock[outIndex+1] = halfToFloatQuick(block[inIndex+1])
			decompressedBlock[outIndex+2] = halfToFloatQuick(block[inIndex+2])
		}
	}
}

// bc7 decodes 16 bytes from compressedBlock to RGBA8
// with destinationPitch many bytes per output row.
func bc7(compressedBlock []byte, decompressedBlock []byte, destinationPitch int) {
	bstream := bitstream{
		low:  binary.LittleEndian.Uint64(compressedBlock[0:8]),
		high: binary.LittleEndian.Uint64(compressedBlock[8:16]),
	}

	var endpoints [6][4]uint32
	var indices [4][4]uint8

	mode := uint32(0)
	for mode < 8 && bstream.readBit() == 0 {
		mode++
	}

	// unexpected mode, clear the block (transparent black)
	if mode >= 8 {
		for i := 0; i < 4; i++ {
			index := i * destinationPitch
			for k := 0; k < 4*4; k++ {
				decompressedBlock[index+k] = 0
			}
		}

		return
	}

	partition := uint32(0)
	numPartitions := 1
	rotation := uint32(0)
	indexSelectionBit := uint32(0)

	if mode == 0 || mode == 1 || mode == 2 || mode == 3 || mode == 7 {
		if mode == 0 || mode == 2 {
			numPartitions = 3
		} else {
			numPartitions = 2
		}
		if mode == 0 {
			partition = bstream.readBits(4)
		} else {
			partition = bstream.readBits(6)
		}
	}

	numEndpoints := numPartitions * 2

	if mode == 4 || mode == 5 {
		rotation = bstream.readBits(2)

		if mode == 4 {
			indexSelectionBit = bstream.readBit()
		}
	}

	// Extract endpoints
	// RGB
	for i := 0; i < 3; i++ {
		for e := 0; e < numEndpoints; e++ {
			endpoints[e][i] = bstream.readBits(uint32(bc7ActualBitsCount[0][mode]))
		}
	}
	// Alpha (if any)
	if bc7ActualBitsCount[1][mode] > 0 {
		for e := 0; e < numEndpoints; e++ {
			endpoints[e][3] = bstream.readBits(uint32(bc7ActualBitsCount[1][mode]))
		}
	}

	// Fully decode endpoints
	// First handle modes that have P-bits
	if mode == 0 || mode == 1 || mode == 3 || mode == 6 || mode == 7 {
		for e := 0; e < numEndpoints; e++ {
			// component-wise left-shift
			for c := 0; c < 4; c++ {
				endpoints[e][c] <<= 1
			}
		}

		// if P-bit is shared
		if mode == 1 {
			i := bstream.readBit()
			j := bstream.readBit()

			// rgb component-wise insert pbits
			for k := 0; k < 3; k++ {
				endpoints[0][k] |= i
				endpoints[1][k] |= i
				endpoints[2][k] |= j
				endpoints[3][k] |= j
			}
		} else if (bc7SModeHasPbits & (uint8(1) << mode)) != 0 {
			// unique P-bit per endpoint
			for e := 0; e < numEndpoints; e++ {
				j := bstream.readBit()
				for c := 0; c < 4; c++ {
					endpoints[e][c] |= j
				}
			}
		}
	}

	for e := 0; e < numEndpoints; e++ {
		// get color components precision including pbit
		j := uint32(bc7ActualBitsCount[0][mode]) + uint32((bc7SModeHasPbits>>mode)&1)

		for c := 0; c < 3; c++ {
			// left shift endpoint components so that their MSB lies in bit 7
			endpoints[e][c] <<= 8 - j
			// Replicate each component's MSB into the LSBs revealed by the left-shift operation above
			endpoints[e][c] |= endpoints[e][c] >> j
		}

		// get alpha component precision including pbit
		j = uint32(bc7ActualBitsCount[1][mode]) + uint32((bc7SModeHasPbits>>mode)&1)

		// left shift endpoint components so that their MSB lies in bit 7
		endpoints[e][3] <<= 8 - j
		// Replicate each component's MSB into the LSBs revealed by the left-shift operation above
		endpoints[e][3] |= endpoints[e][3] >> j
	}

	// If this mode does not explicitly define the alpha component
	// set alpha equal to 1.0
	if bc7ActualBitsCount[1][mode] == 0 {
		for e := 0; e < numEndpoints; e++ {
			endpoints[e][3] = 0xFF
		}
	}

	// Determine weights tables
	indexBits := uint32(2)
	if mode == 0 || mode == 1 {
		indexBits = 3
	} else if mode == 6 {
		indexBits = 4
	}
	indexBits2 := uint32(0)
	if mode == 4 {
		indexBits2 = 3
	} else if mode == 5 {
		indexBits2 = 2
	}
	var weights []uint32
	if indexBits == 2 {
		weights = bc7AWeight2[:]
	} else if indexBits == 3 {
		weights = bc7AWeight3[:]
	} else {
		weights = bc7AWeight4[:]
	}
	var weights2 []uint32
	if indexBits2 == 2 {
		weights2 = bc7AWeight2[:]
	} else {
		weights2 = bc7AWeight3[:]
	}

	// Quite inconvenient that indices aren't interleaved so we have to make 2 passes here
	// Pass #1: collecting color indices
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			var partitionSet uint8
			if numPartitions == 1 {
				if (i | j) != 0 {
					partitionSet = 0
				} else {
					partitionSet = 128
				}
			} else {
				partitionSet = bc7PartitionSets[numPartitions-2][partition][i][j]
			}

			indexBits = 2
			if mode == 0 || mode == 1 {
				indexBits = 3
			} else if mode == 6 {
				indexBits = 4
			}
			// fix-up index is specified with one less bit
			// The fix-up index for subset 0 is always index 0
			if partitionSet&0x80 != 0 {
				indexBits--
			}

			indices[i][j] = uint8(bstream.readBits(indexBits))
		}
	}

	// Pass #2: reading alpha indices (if any) and interpolating & rotating
	for i := 0; i < 4; i++ {
		if len(decompressedBlock) < i*destinationPitch+4*4 {
			panic("bc7: decompressedBlock is too small")
		}

		for j := 0; j < 4; j++ {
			partitionSet := 0
			if numPartitions == 1 {
				if (i | j) != 0 {
					partitionSet = 0
				} else {
					partitionSet = 128
				}
			} else {
				partitionSet = int(bc7PartitionSets[numPartitions-2][partition][i][j])
			}
			partitionSet &= 0x03

			weight := weights[indices[i][j]]

			var r, g, b, a uint32
			if indexBits2 == 0 {
				r = interpolate(endpoints[partitionSet*2][0], endpoints[partitionSet*2+1][0], weight)
				g = interpolate(endpoints[partitionSet*2][1], endpoints[partitionSet*2+1][1], weight)
				b = interpolate(endpoints[partitionSet*2][2], endpoints[partitionSet*2+1][2], weight)
				a = interpolate(endpoints[partitionSet*2][3], endpoints[partitionSet*2+1][3], weight)
			} else {
				n := indexBits2
				if (i | j) == 0 {
					n = indexBits2 - 1
				}
				index2 := bstream.readBits(n)
				weight2 := weights2[index2]

				// The index value for interpolating color comes from the secondary index bits for the texel
				// if the mode has an index selection bit and its value is one, and from the primary index bits otherwise.
				// The alpha index comes from the secondary index bits if the block has a secondary index and
				// the block either doesn't have an index selection bit or that bit is zero, and from the primary index bits otherwise.
				if indexSelectionBit == 0 {
					r = interpolate(endpoints[partitionSet*2][0], endpoints[partitionSet*2+1][0], weight)
					g = interpolate(endpoints[partitionSet*2][1], endpoints[partitionSet*2+1][1], weight)
					b = interpolate(endpoints[partitionSet*2][2], endpoints[partitionSet*2+1][2], weight)
					a = interpolate(endpoints[partitionSet*2][3], endpoints[partitionSet*2+1][3], weight2)
				} else {
					r = interpolate(endpoints[partitionSet*2][0], endpoints[partitionSet*2+1][0], weight2)
					g = interpolate(endpoints[partitionSet*2][1], endpoints[partitionSet*2+1][1], weight2)
					b = interpolate(endpoints[partitionSet*2][2], endpoints[partitionSet*2+1][2], weight2)
					a = interpolate(endpoints[partitionSet*2][3], endpoints[partitionSet*2+1][3], weight)
				}
			}

			switch rotation {
			case 1:
				// 01 – Block format is Scalar(R) Vector(AGB) - swap A and R
				a, r = r, a
			case 2:
				// 10 – Block format is Scalar(G) Vector(RAB) - swap A and G
				a, g = g, a
			case 3:
				// 11 - Block format is Scalar(B) Vector(RGA) - swap A and B
				a, b = b, a
			}

			index := i*destinationPitch + j*4
			decompressedBlock[index] = uint8(r)
			decompressedBlock[index+1] = uint8(g)
			decompressedBlock[index+2] = uint8(b)
			decompressedBlock[index+3] = uint8(a)
		}
	}
}

func colorBlock(compressedBlock []byte, decompressedBlock []byte, destinationPitch int, onlyOpaqueMode bool) {
	var refColors [4][4]byte // 0xAABBGGRR

	c0 := binary.LittleEndian.Uint16(compressedBlock[0:2])
	c1 := binary.LittleEndian.Uint16(compressedBlock[2:4])

	// Unpack 565 ref colors
	r0 := uint32(c0>>11) & 0x1F
	g0 := uint32(c0>>5) & 0x3F
	b0 := uint32(c0) & 0x1F

	r1 := uint32(c1>>11) & 0x1F
	g1 := uint32(c1>>5) & 0x3F
	b1 := uint32(c1) & 0x1F

	// Expand 565 ref colors to 888
	r := (r0*527 + 23) >> 6
	g := (g0*259 + 33) >> 6
	b := (b0*527 + 23) >> 6
	refColors[0] = [4]byte{uint8(r), uint8(g), uint8(b), 255}

	r = (r1*527 + 23) >> 6
	g = (g1*259 + 33) >> 6
	b = (b1*527 + 23) >> 6
	refColors[1] = [4]byte{uint8(r), uint8(g), uint8(b), 255}

	if c0 > c1 || onlyOpaqueMode {
		// Standard BC1 mode (also BC3 color block uses ONLY this mode)
		// color_2 = 2/3*color_0 + 1/3*color_1
		// color_3 = 1/3*color_0 + 2/3*color_1
		r = ((2*r0+r1)*351 + 61) >> 7
		g = ((2*g0+g1)*2763 + 1039) >> 11
		b = ((2*b0+b1)*351 + 61) >> 7
		refColors[2] = [4]byte{uint8(r), uint8(g), uint8(b), 255}

		r = ((r0+r1*2)*351 + 61) >> 7
		g = ((g0+g1*2)*2763 + 1039) >> 11
		b = ((b0+b1*2)*351 + 61) >> 7
		refColors[3] = [4]byte{uint8(r), uint8(g), uint8(b), 255}
	} else {
		// Quite rare BC1A mode
		// color_2 = 1/2*color_0 + 1/2*color_1;
		// color_3 = 0;
		r = ((r0+r1)*1053 + 125) >> 8
		g = ((g0+g1)*4145 + 1019) >> 11
		b = ((b0+b1)*1053 + 125) >> 8
		refColors[2] = [4]byte{uint8(r), uint8(g), uint8(b), 255}

		refColors[3] = [4]byte{0, 0, 0, 0}
	}

	colorIndices := binary.LittleEndian.Uint32(compressedBlock[4:8])

	// Fill out the decompressed color block
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			idx := colorIndices & 0x03
			start := i*destinationPitch + j*4
			copy(decompressedBlock[start:start+4], refColors[idx][:])
			colorIndices >>= 2
		}
	}
}

func sharpAlphaBlock(compressedBlock []byte, decompressedBlock []byte, destinationPitch int) {
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			index := i*destinationPitch + j*4 + 3
			alpha := binary.LittleEndian.Uint16(compressedBlock[i*2 : i*2+2])
			decompressedBlock[index] = uint8((alpha>>(4*j))&0x0F) * 17
		}
	}
}

func smoothAlphaBlock(compressedBlock []byte, decompressedBlock []byte, destinationPitch int, pixelSize int) {
	var alpha [8]uint32

	alpha[0] = uint32(compressedBlock[0])
	alpha[1] = uint32(compressedBlock[1])

	if alpha[0] > alpha[1] {
		// 6 interpolated alpha values.
		alpha[2] = (6*alpha[0] + alpha[1] + 1) / 7   // 6/7*alpha_0 + 1/7*alpha_1
		alpha[3] = (5*alpha[0] + 2*alpha[1] + 1) / 7 // 5/7*alpha_0 + 2/7*alpha_1
		alpha[4] = (4*alpha[0] + 3*alpha[1] + 1) / 7 // 4/7*alpha_0 + 3/7*alpha_1
		alpha[5] = (3*alpha[0] + 4*alpha[1] + 1) / 7 // 3/7*alpha_0 + 4/7*alpha_1
		alpha[6] = (2*alpha[0] + 5*alpha[1] + 1) / 7 // 2/7*alpha_0 + 5/7*alpha_1
		alpha[7] = (alpha[0] + 6*alpha[1] + 1) / 7   // 1/7*alpha_0 + 6/7*alpha_1
	} else {
		// 4 interpolated alpha values.
		alpha[2] = (4*alpha[0] + alpha[1] + 1) / 5   // 4/5*alpha_0 + 1/5*alpha_1
		alpha[3] = (3*alpha[0] + 2*alpha[1] + 1) / 5 // 3/5*alpha_0 + 2/5*alpha_1
		alpha[4] = (2*alpha[0] + 3*alpha[1] + 1) / 5 // 2/5*alpha_0 + 3/5*alpha_1
		alpha[5] = (alpha[0] + 4*alpha[1] + 1) / 5   // 1/5*alpha_0 + 4/5*alpha_1
		alpha[6] = 0x00
		alpha[7] = 0xFF
	}

	block := binary.LittleEndian.Uint64(compressedBlock[:8])
	indices := block >> 16
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			index := i*destinationPitch + j*pixelSize
			decompressedBlock[index] = uint8(alpha[indices&0x07])
			indices >>= 3
		}
	}
}

func bc4Block(compressedBlock []byte, decompressedBlock []byte, destinationPitch int, pixelSize int, isSigned bool) {
	var alpha [8]int32

	if isSigned {
		alpha[0] = int32(int8(compressedBlock[0]))
		alpha[1] = int32(int8(compressedBlock[1]))
		if alpha[0] < -127 {
			// -128 clamps to -127
			alpha[0] = -127
		}
		if alpha[1] < -127 {
			// -128 clamps to -127
			alpha[1] = -127
		}
	} else {
		alpha[0] = int32(compressedBlock[0])
		alpha[1] = int32(compressedBlock[1])
	}

	if alpha[0] > alpha[1] {
		// 6 interpolated alpha values.
		alpha[2] = (bc4AWeights6[5]*alpha[0] + bc4AWeights6[0]*alpha[1] + 32768) >> 16 // 6/7*alpha_0 + 1/7*alpha_1
		alpha[3] = (bc4AWeights6[4]*alpha[0] + bc4AWeights6[1]*alpha[1] + 32768) >> 16 // 5/7*alpha_0 + 2/7*alpha_1
		alpha[4] = (bc4AWeights6[3]*alpha[0] + bc4AWeights6[2]*alpha[1] + 32768) >> 16 // 4/7*alpha_0 + 3/7*alpha_1
		alpha[5] = (bc4AWeights6[2]*alpha[0] + bc4AWeights6[3]*alpha[1] + 32768) >> 16 // 3/7*alpha_0 + 4/7*alpha_1
		alpha[6] = (bc4AWeights6[1]*alpha[0] + bc4AWeights6[4]*alpha[1] + 32768) >> 16 // 2/7*alpha_0 + 5/7*alpha_1
		alpha[7] = (bc4AWeights6[0]*alpha[0] + bc4AWeights6[5]*alpha[1] + 32768) >> 16 // 1/7*alpha_0 + 6/7*alpha_1
	} else {
		// 4 interpolated alpha values.
		alpha[2] = (bc4AWeights4[3]*alpha[0] + bc4AWeights4[0]*alpha[1] + 32768) >> 16 // 4/5*alpha_0 + 1/5*alpha_1
		alpha[3] = (bc4AWeights4[2]*alpha[0] + bc4AWeights4[1]*alpha[1] + 32768) >> 16 // 3/5*alpha_0 + 2/5*alpha_1
		alpha[4] = (bc4AWeights4[1]*alpha[0] + bc4AWeights4[2]*alpha[1] + 32768) >> 16 // 2/5*alpha_0 + 3/5*alpha_1
		alpha[5] = (bc4AWeights4[0]*alpha[0] + bc4AWeights4[3]*alpha[1] + 32768) >> 16 // 1/5*alpha_0 + 4/5*alpha_1
		if isSigned {
			alpha[6] = -127
			alpha[7] = 127
		} else {
			alpha[6] = 0
			alpha[7] = 255
		}
	}

	block := binary.LittleEndian.Uint64(compressedBlock[:8])
	indices := block >> 16
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			index := i*destinationPitch + j*pixelSize
			decompressedBlock[index] = uint8(alpha[indices&0x7])
			indices >>= 3
		}
	}
}

func bc4BlockFloat(compressedBlock []byte, decompressedBlock []float32, destinationPitch int, pixelSize int, isSigned bool) {
	var alpha [8]float32

	if isSigned {
		alpha[0] = float32(int8(compressedBlock[0])) / 127.0
		alpha[1] = float32(int8(compressedBlock[1])) / 127.0
		if alpha[0] < -1.0 {
			// -128 clamps to -127
			alpha[0] = -1.0
		}
		if alpha[1] < -1.0 {
			// -128 clamps to -127
			alpha[1] = -1.0
		}
	} else {
		alpha[0] = float32(compressedBlock[0]) / 255.0
		alpha[1] = float32(compressedBlock[1]) / 255.0
	}

	if alpha[0] > alpha[1] {
		// 6 interpolated alpha values.
		alpha[2] = (6.0*alpha[0] + alpha[1]) / 7.0     // 6/7*alpha_0 + 1/7*alpha_1
		alpha[3] = (5.0*alpha[0] + 2.0*alpha[1]) / 7.0 // 5/7*alpha_0 + 2/7*alpha_1
		alpha[4] = (4.0*alpha[0] + 3.0*alpha[1]) / 7.0 // 4/7*alpha_0 + 3/7*alpha_1
		alpha[5] = (3.0*alpha[0] + 4.0*alpha[1]) / 7.0 // 3/7*alpha_0 + 4/7*alpha_1
		alpha[6] = (2.0*alpha[0] + 5.0*alpha[1]) / 7.0 // 2/7*alpha_0 + 5/7*alpha_1
		alpha[7] = (alpha[0] + 6.0*alpha[1]) / 7.0     // 1/7*alpha_0 + 6/7*alpha_1
	} else {
		// 4 interpolated alpha values.
		alpha[2] = (4.0*alpha[0] + alpha[1]) / 5.0     // 4/5*alpha_0 + 1/5*alpha_1
		alpha[3] = (3.0*alpha[0] + 2.0*alpha[1]) / 5.0 // 3/5*alpha_0 + 2/5*alpha_1
		alpha[4] = (2.0*alpha[0] + 3.0*alpha[1]) / 5.0 // 2/5*alpha_0 + 3/5*alpha_1
		alpha[5] = (alpha[0] + 4.0*alpha[1]) / 5.0     // 1/5*alpha_0 + 4/5*alpha_1
		if isSigned {
			alpha[6] = -1.0
		} else {
			alpha[6] = 0.0
		}
		alpha[7] = 1.0
	}

	block := binary.LittleEndian.Uint64(compressedBlock[:8])
	indices := block >> 16
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			index := i*destinationPitch + j*pixelSize
			decompressedBlock[index] = alpha[indices&0x7]
			indices >>= 3
		}
	}
}

// bitstream is a little-endian bit reader over two 64-bit words.
type bitstream struct {
	low  uint64
	high uint64
}

func (b *bitstream) readBits(numBits uint32) uint32 {
	mask := (uint64(1) << numBits) - 1
	// Read the low N bits
	bits := b.low & mask

	b.low >>= numBits & 63
	// Put the low N bits of "high" into the high 64-N bits of "low".
	b.low |= (b.high & mask) << ((64 - numBits) & 63)
	b.high >>= numBits & 63

	return uint32(bits)
}

func (b *bitstream) readBit() uint32 {
	return b.readBits(1)
}

func (b *bitstream) readBitsI32(numBits uint32) int32 {
	return int32(b.readBits(numBits))
}

func (b *bitstream) readBitI32() int32 {
	return int32(b.readBits(1))
}

// readBitsR pulls bits in reversed order, used in BC6H decoding.
// why ?? just why ???
func (b *bitstream) readBitsR(numBits uint32) int32 {
	bits := b.readBitsI32(numBits)
	// Reverse the bits.
	var result int32
	for i := uint32(0); i < numBits; i++ {
		result <<= 1
		result |= bits & 1
		bits >>= 1
	}
	return result
}

func extendSign(val int32, bits int32) int32 {
	return (val << (32 - bits)) >> (32 - bits)
}

func transformInverse(val int32, a0 int32, bits int32, isSigned bool) int32 {
	// If the precision of A0 is "p" bits, then the transform algorithm is:
	// B0 = (B0 + A0) & ((1 << p) - 1)
	val = (val + a0) & ((int32(1) << bits) - 1)
	if isSigned {
		val = extendSign(val, bits)
	}
	return val
}

// pretty much copy-paste from documentation
func unquantize(val int32, bits int32, isSigned bool) int32 {
	var unq int32
	s := int32(0)

	if !isSigned {
		if bits >= 15 {
			unq = val
		} else if val == 0 {
			unq = 0
		} else if val == (int32(1)<<bits)-1 {
			unq = 0xFFFF
		} else {
			unq = ((val << 16) + 0x8000) >> bits
		}
	} else {
		if bits >= 16 {
			// (Dead code in practice: the original C code assigned unq = val here.)
		} else if val < 0 {
			s = 1
			val = -val
		}

		if val == 0 {
			unq = 0
		} else if val >= (int32(1)<<(bits-1))-1 {
			unq = 0x7FFF
		} else {
			unq = ((val << 15) + 0x4000) >> (bits - 1)
		}

		if s != 0 {
			unq = -unq
		}
	}
	return unq
}

func interpolate(a uint32, b uint32, weight uint32) uint32 {
	return (a*(64-weight) + b*weight + 32) >> 6
}

func interpolateI32(a int32, b int32, weight int32) int32 {
	return (a*(64-weight) + b*weight + 32) >> 6
}

func finishUnquantize(val int32, isSigned bool) uint16 {
	if !isSigned {
		return uint16((val * 31) >> 6) // scale the magnitude by 31 / 64
	}
	v := val
	if v < 0 {
		v = -((-v * 31) >> 5)
	} else {
		v = (v * 31) >> 5
	} // scale the magnitude by 31 / 32
	var s int32
	if v < 0 {
		s = 0x8000
		v = -v
	}
	return uint16(s | v)
}

// halfToFloatQuick is a modified half_to_float_fast4 from
// https://gist.github.com/rygorous/2144712
func halfToFloatQuick(half uint16) float32 {
	magic := math.Float32frombits(113 << 23)
	shiftedExp := uint32(0x7c00) << 13 // exponent mask after shift

	o := uint32(half&0x7fff) << 13 // exponent/mantissa bits
	exp := shiftedExp & o          // just the exponent
	o += (127 - 15) << 23          // exponent adjust

	// handle exponent special cases
	if exp == shiftedExp {
		// Inf/NaN?
		o += (128 - 16) << 23 // extra exp adjust
	} else if exp == 0 {
		// Zero/Denormal?
		o += 1 << 23                                          // extra exp adjust
		o = math.Float32bits(math.Float32frombits(o) - magic) // renormalize
	}

	o |= uint32(half&0x8000) << 16 // sign bit
	return math.Float32frombits(o)
}
