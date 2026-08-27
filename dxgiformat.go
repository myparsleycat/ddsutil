package ddsutil

// The MIT License (MIT)
//
// Copyright (c) 2018 Michael Dilger
//
// ... (see LICENSE for the full text)

import "fmt"

// DxgiFormat is a DXGI pixel format, matching the values of the DXGI_FORMAT
// enum used by the DX10 DDS header extension.
type DxgiFormat uint32

// DXGI formats.
const (
	DxgiFormatUnknown                    DxgiFormat = 0
	DxgiFormatR32G32B32A32_Typeless      DxgiFormat = 1
	DxgiFormatR32G32B32A32_Float         DxgiFormat = 2
	DxgiFormatR32G32B32A32_UInt          DxgiFormat = 3
	DxgiFormatR32G32B32A32_SInt          DxgiFormat = 4
	DxgiFormatR32G32B32_Typeless         DxgiFormat = 5
	DxgiFormatR32G32B32_Float            DxgiFormat = 6
	DxgiFormatR32G32B32_UInt             DxgiFormat = 7
	DxgiFormatR32G32B32_SInt             DxgiFormat = 8
	DxgiFormatR16G16B16A16_Typeless      DxgiFormat = 9
	DxgiFormatR16G16B16A16_Float         DxgiFormat = 10
	DxgiFormatR16G16B16A16_UNorm         DxgiFormat = 11
	DxgiFormatR16G16B16A16_UInt          DxgiFormat = 12
	DxgiFormatR16G16B16A16_SNorm         DxgiFormat = 13
	DxgiFormatR16G16B16A16_SInt          DxgiFormat = 14
	DxgiFormatR32G32_Typeless            DxgiFormat = 15
	DxgiFormatR32G32_Float               DxgiFormat = 16
	DxgiFormatR32G32_UInt                DxgiFormat = 17
	DxgiFormatR32G32_SInt                DxgiFormat = 18
	DxgiFormatR32G8X24_Typeless          DxgiFormat = 19
	DxgiFormatD32_Float_S8X24_UInt       DxgiFormat = 20
	DxgiFormatR32_Float_X8X24_Typeless   DxgiFormat = 21
	DxgiFormatX32_Typeless_G8X24_UInt    DxgiFormat = 22
	DxgiFormatR10G10B10A2_Typeless       DxgiFormat = 23
	DxgiFormatR10G10B10A2_UNorm          DxgiFormat = 24
	DxgiFormatR10G10B10A2_UInt           DxgiFormat = 25
	DxgiFormatR11G11B10_Float            DxgiFormat = 26
	DxgiFormatR8G8B8A8_Typeless          DxgiFormat = 27
	DxgiFormatR8G8B8A8_UNorm             DxgiFormat = 28
	DxgiFormatR8G8B8A8_UNorm_sRGB        DxgiFormat = 29
	DxgiFormatR8G8B8A8_UInt              DxgiFormat = 30
	DxgiFormatR8G8B8A8_SNorm             DxgiFormat = 31
	DxgiFormatR8G8B8A8_SInt              DxgiFormat = 32
	DxgiFormatR16G16_Typeless            DxgiFormat = 33
	DxgiFormatR16G16_Float               DxgiFormat = 34
	DxgiFormatR16G16_UNorm               DxgiFormat = 35
	DxgiFormatR16G16_UInt                DxgiFormat = 36
	DxgiFormatR16G16_SNorm               DxgiFormat = 37
	DxgiFormatR16G16_SInt                DxgiFormat = 38
	DxgiFormatR32_Typeless               DxgiFormat = 39
	DxgiFormatD32_Float                  DxgiFormat = 40
	DxgiFormatR32_Float                  DxgiFormat = 41
	DxgiFormatR32_UInt                   DxgiFormat = 42
	DxgiFormatR32_SInt                   DxgiFormat = 43
	DxgiFormatR24G8_Typeless             DxgiFormat = 44
	DxgiFormatD24_UNorm_S8_UInt          DxgiFormat = 45
	DxgiFormatR24_UNorm_X8_Typeless      DxgiFormat = 46
	DxgiFormatX24_Typeless_G8_UInt       DxgiFormat = 47
	DxgiFormatR8G8_Typeless              DxgiFormat = 48
	DxgiFormatR8G8_UNorm                 DxgiFormat = 49
	DxgiFormatR8G8_UInt                  DxgiFormat = 50
	DxgiFormatR8G8_SNorm                 DxgiFormat = 51
	DxgiFormatR8G8_SInt                  DxgiFormat = 52
	DxgiFormatR16_Typeless               DxgiFormat = 53
	DxgiFormatR16_Float                  DxgiFormat = 54
	DxgiFormatD16_UNorm                  DxgiFormat = 55
	DxgiFormatR16_UNorm                  DxgiFormat = 56
	DxgiFormatR16_UInt                   DxgiFormat = 57
	DxgiFormatR16_SNorm                  DxgiFormat = 58
	DxgiFormatR16_SInt                   DxgiFormat = 59
	DxgiFormatR8_Typeless                DxgiFormat = 60
	DxgiFormatR8_UNorm                   DxgiFormat = 61
	DxgiFormatR8_UInt                    DxgiFormat = 62
	DxgiFormatR8_SNorm                   DxgiFormat = 63
	DxgiFormatR8_SInt                    DxgiFormat = 64
	DxgiFormatA8_UNorm                   DxgiFormat = 65
	DxgiFormatR1_UNorm                   DxgiFormat = 66
	DxgiFormatR9G9B9E5_SharedExp         DxgiFormat = 67
	DxgiFormatR8G8_B8G8_UNorm            DxgiFormat = 68
	DxgiFormatG8R8_G8B8_UNorm            DxgiFormat = 69
	DxgiFormatBC1_Typeless               DxgiFormat = 70
	DxgiFormatBC1_UNorm                  DxgiFormat = 71
	DxgiFormatBC1_UNorm_sRGB             DxgiFormat = 72
	DxgiFormatBC2_Typeless               DxgiFormat = 73
	DxgiFormatBC2_UNorm                  DxgiFormat = 74
	DxgiFormatBC2_UNorm_sRGB             DxgiFormat = 75
	DxgiFormatBC3_Typeless               DxgiFormat = 76
	DxgiFormatBC3_UNorm                  DxgiFormat = 77
	DxgiFormatBC3_UNorm_sRGB             DxgiFormat = 78
	DxgiFormatBC4_Typeless               DxgiFormat = 79
	DxgiFormatBC4_UNorm                  DxgiFormat = 80
	DxgiFormatBC4_SNorm                  DxgiFormat = 81
	DxgiFormatBC5_Typeless               DxgiFormat = 82
	DxgiFormatBC5_UNorm                  DxgiFormat = 83
	DxgiFormatBC5_SNorm                  DxgiFormat = 84
	DxgiFormatB5G6R5_UNorm               DxgiFormat = 85
	DxgiFormatB5G5R5A1_UNorm             DxgiFormat = 86
	DxgiFormatB8G8R8A8_UNorm             DxgiFormat = 87
	DxgiFormatB8G8R8X8_UNorm             DxgiFormat = 88
	DxgiFormatR10G10B10_XR_Bias_A2_UNorm DxgiFormat = 89
	DxgiFormatB8G8R8A8_Typeless          DxgiFormat = 90
	DxgiFormatB8G8R8A8_UNorm_sRGB        DxgiFormat = 91
	DxgiFormatB8G8R8X8_Typeless          DxgiFormat = 92
	DxgiFormatB8G8R8X8_UNorm_sRGB        DxgiFormat = 93
	DxgiFormatBC6H_Typeless              DxgiFormat = 94
	DxgiFormatBC6H_UF16                  DxgiFormat = 95
	DxgiFormatBC6H_SF16                  DxgiFormat = 96
	DxgiFormatBC7_Typeless               DxgiFormat = 97
	DxgiFormatBC7_UNorm                  DxgiFormat = 98
	DxgiFormatBC7_UNorm_sRGB             DxgiFormat = 99
	DxgiFormatAYUV                       DxgiFormat = 100
	DxgiFormatY410                       DxgiFormat = 101
	DxgiFormatY416                       DxgiFormat = 102
	DxgiFormatNV12                       DxgiFormat = 103
	DxgiFormatP010                       DxgiFormat = 104
	DxgiFormatP016                       DxgiFormat = 105
	DxgiFormatFormat_420_Opaque          DxgiFormat = 106
	DxgiFormatYUY2                       DxgiFormat = 107
	DxgiFormatY210                       DxgiFormat = 108
	DxgiFormatY216                       DxgiFormat = 109
	DxgiFormatNV11                       DxgiFormat = 110
	DxgiFormatAI44                       DxgiFormat = 111
	DxgiFormatIA44                       DxgiFormat = 112
	DxgiFormatP8                         DxgiFormat = 113
	DxgiFormatA8P8                       DxgiFormat = 114
	DxgiFormatB4G4R4A4_UNorm             DxgiFormat = 115
	DxgiFormatP208                       DxgiFormat = 130
	DxgiFormatV208                       DxgiFormat = 131
	DxgiFormatV408                       DxgiFormat = 132
	DxgiFormatForce_UInt                 DxgiFormat = 0xffffffff
)

var dxgiFormatNames = map[DxgiFormat]string{
	DxgiFormatUnknown:                    "Unknown",
	DxgiFormatR32G32B32A32_Typeless:      "R32G32B32A32_Typeless",
	DxgiFormatR32G32B32A32_Float:         "R32G32B32A32_Float",
	DxgiFormatR32G32B32A32_UInt:          "R32G32B32A32_UInt",
	DxgiFormatR32G32B32A32_SInt:          "R32G32B32A32_SInt",
	DxgiFormatR32G32B32_Typeless:         "R32G32B32_Typeless",
	DxgiFormatR32G32B32_Float:            "R32G32B32_Float",
	DxgiFormatR32G32B32_UInt:             "R32G32B32_UInt",
	DxgiFormatR32G32B32_SInt:             "R32G32B32_SInt",
	DxgiFormatR16G16B16A16_Typeless:      "R16G16B16A16_Typeless",
	DxgiFormatR16G16B16A16_Float:         "R16G16B16A16_Float",
	DxgiFormatR16G16B16A16_UNorm:         "R16G16B16A16_UNorm",
	DxgiFormatR16G16B16A16_UInt:          "R16G16B16A16_UInt",
	DxgiFormatR16G16B16A16_SNorm:         "R16G16B16A16_SNorm",
	DxgiFormatR16G16B16A16_SInt:          "R16G16B16A16_SInt",
	DxgiFormatR32G32_Typeless:            "R32G32_Typeless",
	DxgiFormatR32G32_Float:               "R32G32_Float",
	DxgiFormatR32G32_UInt:                "R32G32_UInt",
	DxgiFormatR32G32_SInt:                "R32G32_SInt",
	DxgiFormatR32G8X24_Typeless:          "R32G8X24_Typeless",
	DxgiFormatD32_Float_S8X24_UInt:       "D32_Float_S8X24_UInt",
	DxgiFormatR32_Float_X8X24_Typeless:   "R32_Float_X8X24_Typeless",
	DxgiFormatX32_Typeless_G8X24_UInt:    "X32_Typeless_G8X24_UInt",
	DxgiFormatR10G10B10A2_Typeless:       "R10G10B10A2_Typeless",
	DxgiFormatR10G10B10A2_UNorm:          "R10G10B10A2_UNorm",
	DxgiFormatR10G10B10A2_UInt:           "R10G10B10A2_UInt",
	DxgiFormatR11G11B10_Float:            "R11G11B10_Float",
	DxgiFormatR8G8B8A8_Typeless:          "R8G8B8A8_Typeless",
	DxgiFormatR8G8B8A8_UNorm:             "R8G8B8A8_UNorm",
	DxgiFormatR8G8B8A8_UNorm_sRGB:        "R8G8B8A8_UNorm_sRGB",
	DxgiFormatR8G8B8A8_UInt:              "R8G8B8A8_UInt",
	DxgiFormatR8G8B8A8_SNorm:             "R8G8B8A8_SNorm",
	DxgiFormatR8G8B8A8_SInt:              "R8G8B8A8_SInt",
	DxgiFormatR16G16_Typeless:            "R16G16_Typeless",
	DxgiFormatR16G16_Float:               "R16G16_Float",
	DxgiFormatR16G16_UNorm:               "R16G16_UNorm",
	DxgiFormatR16G16_UInt:                "R16G16_UInt",
	DxgiFormatR16G16_SNorm:               "R16G16_SNorm",
	DxgiFormatR16G16_SInt:                "R16G16_SInt",
	DxgiFormatR32_Typeless:               "R32_Typeless",
	DxgiFormatD32_Float:                  "D32_Float",
	DxgiFormatR32_Float:                  "R32_Float",
	DxgiFormatR32_UInt:                   "R32_UInt",
	DxgiFormatR32_SInt:                   "R32_SInt",
	DxgiFormatR24G8_Typeless:             "R24G8_Typeless",
	DxgiFormatD24_UNorm_S8_UInt:          "D24_UNorm_S8_UInt",
	DxgiFormatR24_UNorm_X8_Typeless:      "R24_UNorm_X8_Typeless",
	DxgiFormatX24_Typeless_G8_UInt:       "X24_Typeless_G8_UInt",
	DxgiFormatR8G8_Typeless:              "R8G8_Typeless",
	DxgiFormatR8G8_UNorm:                 "R8G8_UNorm",
	DxgiFormatR8G8_UInt:                  "R8G8_UInt",
	DxgiFormatR8G8_SNorm:                 "R8G8_SNorm",
	DxgiFormatR8G8_SInt:                  "R8G8_SInt",
	DxgiFormatR16_Typeless:               "R16_Typeless",
	DxgiFormatR16_Float:                  "R16_Float",
	DxgiFormatD16_UNorm:                  "D16_UNorm",
	DxgiFormatR16_UNorm:                  "R16_UNorm",
	DxgiFormatR16_UInt:                   "R16_UInt",
	DxgiFormatR16_SNorm:                  "R16_SNorm",
	DxgiFormatR16_SInt:                   "R16_SInt",
	DxgiFormatR8_Typeless:                "R8_Typeless",
	DxgiFormatR8_UNorm:                   "R8_UNorm",
	DxgiFormatR8_UInt:                    "R8_UInt",
	DxgiFormatR8_SNorm:                   "R8_SNorm",
	DxgiFormatR8_SInt:                    "R8_SInt",
	DxgiFormatA8_UNorm:                   "A8_UNorm",
	DxgiFormatR1_UNorm:                   "R1_UNorm",
	DxgiFormatR9G9B9E5_SharedExp:         "R9G9B9E5_SharedExp",
	DxgiFormatR8G8_B8G8_UNorm:            "R8G8_B8G8_UNorm",
	DxgiFormatG8R8_G8B8_UNorm:            "G8R8_G8B8_UNorm",
	DxgiFormatBC1_Typeless:               "BC1_Typeless",
	DxgiFormatBC1_UNorm:                  "BC1_UNorm",
	DxgiFormatBC1_UNorm_sRGB:             "BC1_UNorm_sRGB",
	DxgiFormatBC2_Typeless:               "BC2_Typeless",
	DxgiFormatBC2_UNorm:                  "BC2_UNorm",
	DxgiFormatBC2_UNorm_sRGB:             "BC2_UNorm_sRGB",
	DxgiFormatBC3_Typeless:               "BC3_Typeless",
	DxgiFormatBC3_UNorm:                  "BC3_UNorm",
	DxgiFormatBC3_UNorm_sRGB:             "BC3_UNorm_sRGB",
	DxgiFormatBC4_Typeless:               "BC4_Typeless",
	DxgiFormatBC4_UNorm:                  "BC4_UNorm",
	DxgiFormatBC4_SNorm:                  "BC4_SNorm",
	DxgiFormatBC5_Typeless:               "BC5_Typeless",
	DxgiFormatBC5_UNorm:                  "BC5_UNorm",
	DxgiFormatBC5_SNorm:                  "BC5_SNorm",
	DxgiFormatB5G6R5_UNorm:               "B5G6R5_UNorm",
	DxgiFormatB5G5R5A1_UNorm:             "B5G5R5A1_UNorm",
	DxgiFormatB8G8R8A8_UNorm:             "B8G8R8A8_UNorm",
	DxgiFormatB8G8R8X8_UNorm:             "B8G8R8X8_UNorm",
	DxgiFormatR10G10B10_XR_Bias_A2_UNorm: "R10G10B10_XR_Bias_A2_UNorm",
	DxgiFormatB8G8R8A8_Typeless:          "B8G8R8A8_Typeless",
	DxgiFormatB8G8R8A8_UNorm_sRGB:        "B8G8R8A8_UNorm_sRGB",
	DxgiFormatB8G8R8X8_Typeless:          "B8G8R8X8_Typeless",
	DxgiFormatB8G8R8X8_UNorm_sRGB:        "B8G8R8X8_UNorm_sRGB",
	DxgiFormatBC6H_Typeless:              "BC6H_Typeless",
	DxgiFormatBC6H_UF16:                  "BC6H_UF16",
	DxgiFormatBC6H_SF16:                  "BC6H_SF16",
	DxgiFormatBC7_Typeless:               "BC7_Typeless",
	DxgiFormatBC7_UNorm:                  "BC7_UNorm",
	DxgiFormatBC7_UNorm_sRGB:             "BC7_UNorm_sRGB",
	DxgiFormatAYUV:                       "AYUV",
	DxgiFormatY410:                       "Y410",
	DxgiFormatY416:                       "Y416",
	DxgiFormatNV12:                       "NV12",
	DxgiFormatP010:                       "P010",
	DxgiFormatP016:                       "P016",
	DxgiFormatFormat_420_Opaque:          "Format_420_Opaque",
	DxgiFormatYUY2:                       "YUY2",
	DxgiFormatY210:                       "Y210",
	DxgiFormatY216:                       "Y216",
	DxgiFormatNV11:                       "NV11",
	DxgiFormatAI44:                       "AI44",
	DxgiFormatIA44:                       "IA44",
	DxgiFormatP8:                         "P8",
	DxgiFormatA8P8:                       "A8P8",
	DxgiFormatB4G4R4A4_UNorm:             "B4G4R4A4_UNorm",
	DxgiFormatP208:                       "P208",
	DxgiFormatV208:                       "V208",
	DxgiFormatV408:                       "V408",
	DxgiFormatForce_UInt:                 "Force_UInt",
}

func (f DxgiFormat) String() string {
	if s, ok := dxgiFormatNames[f]; ok {
		return s
	}
	return fmt.Sprintf("DxgiFormat(%d)", uint32(f))
}

// DxgiFormatFromU32 converts a raw u32 value to a DxgiFormat, returning false
// if it is not a known format value.
func DxgiFormatFromU32(v uint32) (DxgiFormat, bool) {
	f := DxgiFormat(v)
	_, ok := dxgiFormatNames[f]
	return f, ok
}

// GetPitch implements DataFormat. See
// https://msdn.microsoft.com/en-us/library/bb943991.aspx
func (f DxgiFormat) GetPitch(width uint32) (uint32, bool) {
	if f == DxgiFormatR8G8_B8G8_UNorm || f == DxgiFormatG8R8_G8B8_UNorm {
		return ((width + 1) >> 1) * 4, true
	}
	if bpp, ok := f.GetBitsPerPixel(); ok {
		return (width*uint32(bpp) + 7) / 8, true
	}
	if bs, ok := f.GetBlockSize(); ok {
		w := (width + 3) / 4
		if w < 1 {
			w = 1
		}
		return w * bs, true
	}
	return 0, false
}

// GetPitchHeight implements DataFormat.
func (f DxgiFormat) GetPitchHeight() uint32 {
	bs, ok := f.GetBlockSize()
	return defaultPitchHeight(bs, ok)
}

// GetBitsPerPixel implements DataFormat.
func (f DxgiFormat) GetBitsPerPixel() (uint8, bool) {
	switch f {
	case DxgiFormatUnknown:
		return 0, false

	case DxgiFormatR32G32B32A32_Typeless,
		DxgiFormatR32G32B32A32_Float,
		DxgiFormatR32G32B32A32_UInt,
		DxgiFormatR32G32B32A32_SInt:
		return 128, true

	case DxgiFormatR32G32B32_Typeless,
		DxgiFormatR32G32B32_Float,
		DxgiFormatR32G32B32_UInt,
		DxgiFormatR32G32B32_SInt:
		return 96, true

	case DxgiFormatR16G16B16A16_Typeless,
		DxgiFormatR16G16B16A16_Float,
		DxgiFormatR16G16B16A16_UNorm,
		DxgiFormatR16G16B16A16_UInt,
		DxgiFormatR16G16B16A16_SNorm,
		DxgiFormatR16G16B16A16_SInt,
		DxgiFormatR32G32_Typeless,
		DxgiFormatR32G32_Float,
		DxgiFormatR32G32_UInt,
		DxgiFormatR32G32_SInt,
		DxgiFormatR32G8X24_Typeless,
		DxgiFormatD32_Float_S8X24_UInt,
		DxgiFormatR32_Float_X8X24_Typeless,
		DxgiFormatX32_Typeless_G8X24_UInt:
		return 64, true

	case DxgiFormatR10G10B10A2_Typeless,
		DxgiFormatR10G10B10A2_UNorm,
		DxgiFormatR10G10B10A2_UInt,
		DxgiFormatR11G11B10_Float,
		DxgiFormatR8G8B8A8_Typeless,
		DxgiFormatR8G8B8A8_UNorm,
		DxgiFormatR8G8B8A8_UNorm_sRGB,
		DxgiFormatR8G8B8A8_UInt,
		DxgiFormatR8G8B8A8_SNorm,
		DxgiFormatR8G8B8A8_SInt,
		DxgiFormatR16G16_Typeless,
		DxgiFormatR16G16_Float,
		DxgiFormatR16G16_UNorm,
		DxgiFormatR16G16_UInt,
		DxgiFormatR16G16_SNorm,
		DxgiFormatR16G16_SInt,
		DxgiFormatR32_Typeless,
		DxgiFormatD32_Float,
		DxgiFormatR32_Float,
		DxgiFormatR32_UInt,
		DxgiFormatR32_SInt,
		DxgiFormatR24G8_Typeless,
		DxgiFormatD24_UNorm_S8_UInt,
		DxgiFormatR24_UNorm_X8_Typeless,
		DxgiFormatX24_Typeless_G8_UInt:
		return 32, true

	case DxgiFormatR8G8_Typeless,
		DxgiFormatR8G8_UNorm,
		DxgiFormatR8G8_UInt,
		DxgiFormatR8G8_SNorm,
		DxgiFormatR8G8_SInt,
		DxgiFormatR16_Typeless,
		DxgiFormatR16_Float,
		DxgiFormatD16_UNorm,
		DxgiFormatR16_UNorm,
		DxgiFormatR16_UInt,
		DxgiFormatR16_SNorm,
		DxgiFormatR16_SInt:
		return 16, true

	case DxgiFormatR8_Typeless,
		DxgiFormatR8_UNorm,
		DxgiFormatR8_UInt,
		DxgiFormatR8_SNorm,
		DxgiFormatR8_SInt,
		DxgiFormatA8_UNorm:
		return 8, true

	case DxgiFormatR1_UNorm:
		return 1, true

	case DxgiFormatR9G9B9E5_SharedExp:
		return 32, true

	case DxgiFormatR8G8_B8G8_UNorm, DxgiFormatG8R8_G8B8_UNorm:
		return 16, true

	case DxgiFormatB5G6R5_UNorm, DxgiFormatB5G5R5A1_UNorm:
		return 16, true

	case DxgiFormatB8G8R8A8_UNorm,
		DxgiFormatB8G8R8X8_UNorm,
		DxgiFormatR10G10B10_XR_Bias_A2_UNorm,
		DxgiFormatB8G8R8A8_Typeless,
		DxgiFormatB8G8R8A8_UNorm_sRGB,
		DxgiFormatB8G8R8X8_Typeless,
		DxgiFormatB8G8R8X8_UNorm_sRGB:
		return 32, true

	case DxgiFormatAYUV:
		return 32, true
	case DxgiFormatY410:
		return 10, true
	case DxgiFormatY416:
		return 16, true
	case DxgiFormatNV12:
		return 12, true
	case DxgiFormatP010:
		return 10, true
	case DxgiFormatP016:
		return 16, true
	case DxgiFormatFormat_420_Opaque:
		return 20, true
	case DxgiFormatYUY2:
		return 16, true
	case DxgiFormatY210:
		return 10, true
	case DxgiFormatY216:
		return 16, true
	case DxgiFormatNV11:
		return 11, true
	case DxgiFormatAI44:
		return 44, true
	case DxgiFormatIA44:
		return 44, true
	case DxgiFormatP8:
		return 8, true
	case DxgiFormatA8P8:
		return 16, true
	case DxgiFormatB4G4R4A4_UNorm:
		return 16, true
	case DxgiFormatP208:
		return 8, true
	case DxgiFormatV208:
		return 8, true
	case DxgiFormatV408:
		return 8, true

	default:
		return 0, false
	}
}

// GetBlockSize implements DataFormat.
func (f DxgiFormat) GetBlockSize() (uint32, bool) {
	switch f {
	case DxgiFormatBC1_Typeless, DxgiFormatBC1_UNorm, DxgiFormatBC1_UNorm_sRGB:
		return 8, true

	case DxgiFormatBC2_Typeless,
		DxgiFormatBC2_UNorm,
		DxgiFormatBC2_UNorm_sRGB,
		DxgiFormatBC3_Typeless,
		DxgiFormatBC3_UNorm,
		DxgiFormatBC3_UNorm_sRGB:
		return 16, true

	case DxgiFormatBC4_Typeless, DxgiFormatBC4_UNorm, DxgiFormatBC4_SNorm:
		return 8, true

	case DxgiFormatBC5_Typeless,
		DxgiFormatBC5_UNorm,
		DxgiFormatBC5_SNorm,
		DxgiFormatBC6H_Typeless,
		DxgiFormatBC6H_UF16,
		DxgiFormatBC6H_SF16,
		DxgiFormatBC7_Typeless,
		DxgiFormatBC7_UNorm,
		DxgiFormatBC7_UNorm_sRGB:
		return 16, true

	default:
		return 0, false
	}
}

// GetFourCC implements DataFormat. Note: we never use this. For Dxgi formats,
// we set FourCC to DX10 and set the format in the header10 field. But these
// were the FourCCs that were used prior to the header10 extension to DDS.
func (f DxgiFormat) GetFourCC() (FourCC, bool) {
	switch f {
	case DxgiFormatBC1_UNorm:
		return FourCC(FourCCBC1_UNORM), true
	case DxgiFormatBC2_UNorm:
		return FourCC(FourCCBC2_UNORM), true
	case DxgiFormatBC3_UNorm:
		return FourCC(FourCCBC3_UNORM), true
	case DxgiFormatBC4_UNorm:
		return FourCC(FourCCBC4_UNORM), true
	case DxgiFormatBC4_SNorm:
		return FourCC(FourCCBC4_SNORM), true
	case DxgiFormatBC5_UNorm:
		return FourCC(FourCCBC5_UNORM), true
	case DxgiFormatBC5_SNorm:
		return FourCC(FourCCBC5_SNORM), true
	case DxgiFormatR8G8_B8G8_UNorm:
		return FourCC(FourCCR8G8_B8G8_UNORM), true
	case DxgiFormatG8R8_G8B8_UNorm:
		return FourCC(FourCCG8R8_G8B8_UNORM), true
	case DxgiFormatR16G16B16A16_UNorm:
		return FourCC(FourCCR16G16B16A16_UNORM), true
	case DxgiFormatR16G16B16A16_SNorm:
		return FourCC(FourCCR16G16B16A16_SNORM), true
	case DxgiFormatR16_Float:
		return FourCC(FourCCR16_FLOAT), true
	case DxgiFormatR16G16_Float:
		return FourCC(FourCCR16G16_FLOAT), true
	case DxgiFormatR16G16B16A16_Float:
		return FourCC(FourCCR16G16B16A16_FLOAT), true
	case DxgiFormatR32_Float:
		return FourCC(FourCCR32_FLOAT), true
	case DxgiFormatR32G32_Float:
		return FourCC(FourCCR32G32_FLOAT), true
	case DxgiFormatR32G32B32A32_Float:
		return FourCC(FourCCR32G32B32A32_FLOAT), true
	default:
		return 0, false
	}
}

// RequiresExtension implements DataFormat. sRGB, float, compressed, and
// larger-than-u32 formats all require the DX10 extension.
func (f DxgiFormat) RequiresExtension() bool {
	switch f {
	// Too big, and many are also not maskable types
	case DxgiFormatR32G32B32A32_Typeless,
		DxgiFormatR32G32B32A32_Float,
		DxgiFormatR32G32B32A32_UInt,
		DxgiFormatR32G32B32A32_SInt,
		DxgiFormatR32G32B32_Typeless,
		DxgiFormatR32G32B32_Float,
		DxgiFormatR32G32B32_UInt,
		DxgiFormatR32G32B32_SInt,
		DxgiFormatR16G16B16A16_Typeless,
		DxgiFormatR16G16B16A16_Float,
		DxgiFormatR16G16B16A16_UNorm,
		DxgiFormatR16G16B16A16_UInt,
		DxgiFormatR16G16B16A16_SNorm,
		DxgiFormatR16G16B16A16_SInt,
		DxgiFormatR32G32_Typeless,
		DxgiFormatR32G32_Float,
		DxgiFormatR32G32_UInt,
		DxgiFormatR32G32_SInt,
		DxgiFormatR32G8X24_Typeless,
		DxgiFormatD32_Float_S8X24_UInt,
		DxgiFormatR32_Float_X8X24_Typeless,
		DxgiFormatX32_Typeless_G8X24_UInt,
		// Not maskable types
		DxgiFormatR10G10B10A2_Typeless,
		DxgiFormatR11G11B10_Float,
		DxgiFormatR8G8B8A8_Typeless,
		DxgiFormatR8G8B8A8_UNorm_sRGB,
		DxgiFormatR16G16_Typeless,
		DxgiFormatR16G16_Float,
		DxgiFormatR32_Typeless,
		DxgiFormatD32_Float,
		DxgiFormatR32_Float,
		DxgiFormatR24G8_Typeless,
		DxgiFormatR24_UNorm_X8_Typeless,
		DxgiFormatR8G8_Typeless,
		DxgiFormatR16_Typeless,
		DxgiFormatR16_Float,
		// Not maskable types
		DxgiFormatR8_Typeless,
		// Not maskable types
		DxgiFormatR9G9B9E5_SharedExp,
		// Not maskable types
		DxgiFormatR10G10B10_XR_Bias_A2_UNorm,
		DxgiFormatB8G8R8A8_Typeless,
		DxgiFormatB8G8R8A8_UNorm_sRGB,
		DxgiFormatB8G8R8X8_Typeless,
		DxgiFormatB8G8R8X8_UNorm_sRGB,
		// Channels are not actual rgb
		DxgiFormatAYUV,
		DxgiFormatY410,
		DxgiFormatY416,
		DxgiFormatNV12,
		DxgiFormatP010,
		DxgiFormatP016,
		DxgiFormatFormat_420_Opaque,
		DxgiFormatYUY2,
		DxgiFormatY210,
		DxgiFormatY216,
		DxgiFormatNV11,
		DxgiFormatAI44,
		DxgiFormatIA44,
		DxgiFormatP8,
		DxgiFormatA8P8,
		DxgiFormatP208,
		DxgiFormatV208,
		DxgiFormatV408:
		return true
	default:
		return false
	}
}

// GetMinimumMipmapSizeInBytes implements DataFormat.
func (f DxgiFormat) GetMinimumMipmapSizeInBytes() (uint32, bool) {
	bpp, hasBPP := f.GetBitsPerPixel()
	bs, hasBS := f.GetBlockSize()
	return defaultMinimumMipmapSizeInBytes(bpp, hasBPP, bs, hasBS)
}

// DxgiFormatTryFromPixelFormat attempts to use PixelFormat data (e.g. from
// dds.Header.SPF) to determine the DxgiFormat.
func DxgiFormatTryFromPixelFormat(pixelFormat *PixelFormat) (DxgiFormat, bool) {
	if pixelFormat.FourCC != nil {
		switch *pixelFormat.FourCC {
		case FourCCDXT1:
			return DxgiFormatBC1_UNorm_sRGB, true
		case FourCCDXT3:
			return DxgiFormatBC2_UNorm_sRGB, true
		case FourCCDXT5:
			return DxgiFormatBC3_UNorm_sRGB, true
		case FourCCATI1:
			return DxgiFormatBC4_UNorm, true
		case FourCCATI2:
			return DxgiFormatBC5_UNorm, true
		}
	}
	return 0, false
}
