package ddsutil

import (
	"encoding/binary"
	"math"
)

// rgbaFormat describes the byte layout of an uncompressed pixel format.
type rgbaFormat uint8

const (
	formatR8          rgbaFormat = iota // 1 x u8
	formatR8Snorm                       // 1 x i8
	formatRg8                           // 2 x u8
	formatRg8Snorm                      // 2 x i8
	formatRgb8                          // 3 x u8
	formatRgba8                         // 4 x u8
	formatRgba8Snorm                    // 4 x i8
	formatR16                           // 1 x u16
	formatR16Snorm                      // 1 x i16
	formatRg16                          // 2 x u16
	formatRg16Snorm                     // 2 x i16
	formatRgba16                        // 4 x u16
	formatRgba16Snorm                   // 4 x i16
	formatRf16                          // 1 x half
	formatRgf16                         // 2 x half
	formatRgbaf16                       // 4 x half
	formatRf32                          // 1 x f32
	formatRgf32                         // 2 x f32
	formatRgbf32                        // 3 x f32
	formatRgbaf32                       // 4 x f32
	formatBgr8                          // 3 bytes in BGR order
	formatBgra8                         // 4 bytes in BGRA order
	formatBgra4                         // 2 bytes packed A4R4G4B4
	formatBgr5A1                        // 2 bytes packed B5G5R5A1
)

// pixelSize returns the size of a pixel in bytes.
func (f rgbaFormat) pixelSize() int {
	switch f {
	case formatR8, formatR8Snorm:
		return 1
	case formatRg8, formatRg8Snorm, formatR16, formatR16Snorm, formatRf16, formatRf32, formatBgra4, formatBgr5A1:
		return 2
	case formatRgb8, formatBgr8:
		return 3
	case formatRgba8, formatRgba8Snorm, formatRg16, formatRg16Snorm, formatRgf16, formatRgf32, formatBgra8:
		return 4
	case formatRgba16, formatRgba16Snorm, formatRgbaf16:
		return 8
	case formatRgbf32:
		return 12
	case formatRgbaf32:
		return 16
	}
	return 0
}

// validateLength checks that data has at least width*height*elementsPerPixel
// elements, returning the expected count or an error.
func validateLength(width, height uint32, elementsPerPixel int, dataLen int) (int, error) {
	expected, ok := mulOverflow(int(width), int(height))
	if !ok {
		return 0, &SurfaceError{Kind: SurfaceErrorPixelCountWouldOverflow, Width: width, Height: height, Depth: 1}
	}
	expected, ok = mulOverflow(expected, elementsPerPixel)
	if !ok {
		return 0, &SurfaceError{Kind: SurfaceErrorPixelCountWouldOverflow, Width: width, Height: height, Depth: 1}
	}
	if dataLen < expected {
		return 0, &SurfaceError{Kind: SurfaceErrorNotEnoughData, Expected: expected, Actual: dataLen}
	}
	return expected, nil
}

// decodeRgbaU8 decodes pixels of the given format to RGBA8.
func decodeRgbaU8(format rgbaFormat, width, height uint32, data []byte) ([]uint8, error) {
	if _, err := validateLength(width, height, format.pixelSize(), len(data)); err != nil {
		return nil, err
	}
	out := make([]uint8, int(width)*int(height)*4)
	o := 0
	switch format {
	case formatR8:
		for i := 0; i < len(data); i++ {
			v := data[i]
			out[o] = v
			out[o+1] = v
			out[o+2] = v
			out[o+3] = 255
			o += 4
		}
	case formatR8Snorm:
		for i := 0; i < len(data); i++ {
			v := snorm8ToUnorm8(data[i])
			out[o] = v
			out[o+1] = v
			out[o+2] = v
			out[o+3] = 255
			o += 4
		}
	case formatRg8:
		for i := 0; i < len(data); i += 2 {
			out[o] = data[i]
			out[o+1] = data[i+1]
			out[o+2] = 0
			out[o+3] = 255
			o += 4
		}
	case formatRg8Snorm:
		for i := 0; i < len(data); i += 2 {
			out[o] = snorm8ToUnorm8(data[i])
			out[o+1] = snorm8ToUnorm8(data[i+1])
			// The blue channel converts 0 for unorm/snorm by convention.
			out[o+2] = snorm8ToUnorm8(0)
			out[o+3] = 255
			o += 4
		}
	case formatRgb8:
		for i := 0; i < len(data); i += 3 {
			out[o] = data[i]
			out[o+1] = data[i+1]
			out[o+2] = data[i+2]
			out[o+3] = 255
			o += 4
		}
	case formatRgba8:
		copy(out, data)
	case formatRgba8Snorm:
		for i := 0; i < len(data); i += 4 {
			out[o] = snorm8ToUnorm8(data[i])
			out[o+1] = snorm8ToUnorm8(data[i+1])
			out[o+2] = snorm8ToUnorm8(data[i+2])
			out[o+3] = snorm8ToUnorm8(data[i+3])
			o += 4
		}
	case formatR16:
		for i := 0; i < len(data); i += 2 {
			v := unorm16ToUnorm8(binary.LittleEndian.Uint16(data[i:]))
			out[o] = v
			out[o+1] = v
			out[o+2] = v
			out[o+3] = 255
			o += 4
		}
	case formatR16Snorm:
		for i := 0; i < len(data); i += 2 {
			v := snorm16ToUnorm8(binary.LittleEndian.Uint16(data[i:]))
			out[o] = v
			out[o+1] = v
			out[o+2] = v
			out[o+3] = 255
			o += 4
		}
	case formatRg16:
		for i := 0; i < len(data); i += 4 {
			out[o] = unorm16ToUnorm8(binary.LittleEndian.Uint16(data[i:]))
			out[o+1] = unorm16ToUnorm8(binary.LittleEndian.Uint16(data[i+2:]))
			out[o+2] = 0
			out[o+3] = 255
			o += 4
		}
	case formatRg16Snorm:
		for i := 0; i < len(data); i += 4 {
			out[o] = snorm16ToUnorm8(binary.LittleEndian.Uint16(data[i:]))
			out[o+1] = snorm16ToUnorm8(binary.LittleEndian.Uint16(data[i+2:]))
			// The blue channel converts 0 for unorm/snorm by convention.
			out[o+2] = snorm16ToUnorm8(0)
			out[o+3] = 255
			o += 4
		}
	case formatRgba16:
		for i := 0; i < len(data); i += 8 {
			out[o] = unorm16ToUnorm8(binary.LittleEndian.Uint16(data[i:]))
			out[o+1] = unorm16ToUnorm8(binary.LittleEndian.Uint16(data[i+2:]))
			out[o+2] = unorm16ToUnorm8(binary.LittleEndian.Uint16(data[i+4:]))
			out[o+3] = unorm16ToUnorm8(binary.LittleEndian.Uint16(data[i+6:]))
			o += 4
		}
	case formatRgba16Snorm:
		for i := 0; i < len(data); i += 8 {
			out[o] = snorm16ToUnorm8(binary.LittleEndian.Uint16(data[i:]))
			out[o+1] = snorm16ToUnorm8(binary.LittleEndian.Uint16(data[i+2:]))
			out[o+2] = snorm16ToUnorm8(binary.LittleEndian.Uint16(data[i+4:]))
			out[o+3] = snorm16ToUnorm8(binary.LittleEndian.Uint16(data[i+6:]))
			o += 4
		}
	case formatRf16:
		for i := 0; i < len(data); i += 2 {
			v := f32ToU8(halfToFloat32(binary.LittleEndian.Uint16(data[i:])) * 255.0)
			out[o] = v
			out[o+1] = v
			out[o+2] = v
			out[o+3] = 255
			o += 4
		}
	case formatRgf16:
		for i := 0; i < len(data); i += 4 {
			out[o] = f32ToU8(halfToFloat32(binary.LittleEndian.Uint16(data[i:])) * 255.0)
			out[o+1] = f32ToU8(halfToFloat32(binary.LittleEndian.Uint16(data[i+2:])) * 255.0)
			out[o+2] = 0
			out[o+3] = 255
			o += 4
		}
	case formatRgbaf16:
		for i := 0; i < len(data); i += 8 {
			out[o] = f32ToU8(halfToFloat32(binary.LittleEndian.Uint16(data[i:])) * 255.0)
			out[o+1] = f32ToU8(halfToFloat32(binary.LittleEndian.Uint16(data[i+2:])) * 255.0)
			out[o+2] = f32ToU8(halfToFloat32(binary.LittleEndian.Uint16(data[i+4:])) * 255.0)
			out[o+3] = f32ToU8(halfToFloat32(binary.LittleEndian.Uint16(data[i+6:])) * 255.0)
			o += 4
		}
	case formatRf32:
		for i := 0; i < len(data); i += 4 {
			v := f32ToU8(mathFloat32frombits(binary.LittleEndian.Uint32(data[i:])) * 255.0)
			out[o] = v
			out[o+1] = v
			out[o+2] = v
			out[o+3] = 255
			o += 4
		}
	case formatRgf32:
		for i := 0; i < len(data); i += 8 {
			out[o] = f32ToU8(mathFloat32frombits(binary.LittleEndian.Uint32(data[i:])) * 255.0)
			out[o+1] = f32ToU8(mathFloat32frombits(binary.LittleEndian.Uint32(data[i+4:])) * 255.0)
			out[o+2] = 0
			out[o+3] = 255
			o += 4
		}
	case formatRgbf32:
		for i := 0; i < len(data); i += 12 {
			out[o] = f32ToU8(mathFloat32frombits(binary.LittleEndian.Uint32(data[i:])) * 255.0)
			out[o+1] = f32ToU8(mathFloat32frombits(binary.LittleEndian.Uint32(data[i+4:])) * 255.0)
			out[o+2] = f32ToU8(mathFloat32frombits(binary.LittleEndian.Uint32(data[i+8:])) * 255.0)
			out[o+3] = 255
			o += 4
		}
	case formatRgbaf32:
		for i := 0; i < len(data); i += 16 {
			out[o] = f32ToU8(mathFloat32frombits(binary.LittleEndian.Uint32(data[i:])) * 255.0)
			out[o+1] = f32ToU8(mathFloat32frombits(binary.LittleEndian.Uint32(data[i+4:])) * 255.0)
			out[o+2] = f32ToU8(mathFloat32frombits(binary.LittleEndian.Uint32(data[i+8:])) * 255.0)
			out[o+3] = f32ToU8(mathFloat32frombits(binary.LittleEndian.Uint32(data[i+12:])) * 255.0)
			o += 4
		}
	case formatBgr8:
		for i := 0; i < len(data); i += 3 {
			out[o] = data[i+2]
			out[o+1] = data[i+1]
			out[o+2] = data[i]
			out[o+3] = 255
			o += 4
		}
	case formatBgra8:
		for i := 0; i < len(data); i += 4 {
			out[o] = data[i+2]
			out[o+1] = data[i+1]
			out[o+2] = data[i]
			out[o+3] = data[i+3]
			o += 4
		}
	case formatBgra4:
		for i := 0; i < len(data); i += 2 {
			// Most significant bit -> AAAARRRRGGGGBBBB -> least significant bit.
			out[o] = unorm4ToUnorm8(data[i+1] & 0xF)
			out[o+1] = unorm4ToUnorm8(data[i] >> 4)
			out[o+2] = unorm4ToUnorm8(data[i] & 0xF)
			out[o+3] = unorm4ToUnorm8(data[i+1] >> 4)
			o += 4
		}
	case formatBgr5A1:
		for i := 0; i < len(data); i += 2 {
			bytes := binary.BigEndian.Uint16(data[i:])
			out[o] = unorm5ToUnorm8(uint8((bytes >> 2) & 0x1F))
			out[o+1] = unorm5ToUnorm8(uint8(rotateLeft16(bytes, 3) & 0x1F))
			out[o+2] = unorm5ToUnorm8(uint8((bytes >> 8) & 0x1F))
			if (bytes>>7)&0x1 == 1 {
				out[o+3] = 255
			} else {
				out[o+3] = 0
			}
			o += 4
		}
	}
	return out, nil
}

// decodeRgbaF32 decodes pixels of the given format to RGBAF32. Formats that
// do not store floating point data fall back to the u8 decoding divided
// by 255.
func decodeRgbaF32(format rgbaFormat, width, height uint32, data []byte) ([]float32, error) {
	switch format {
	case formatR8Snorm, formatRg8Snorm, formatRgba8Snorm,
		formatRf16, formatRgf16, formatRgbaf16,
		formatRf32, formatRgf32, formatRgbf32, formatRgbaf32,
		formatR16, formatR16Snorm, formatRg16, formatRg16Snorm,
		formatRgba16, formatRgba16Snorm:
		// handled below
	default:
		// Use existing decoding for formats that don't store floating point data.
		rgba8, err := decodeRgbaU8(format, width, height, data)
		if err != nil {
			return nil, err
		}
		out := make([]float32, len(rgba8))
		for i, u := range rgba8 {
			out[i] = float32(u) / 255.0
		}
		return out, nil
	}

	if _, err := validateLength(width, height, format.pixelSize(), len(data)); err != nil {
		return nil, err
	}
	out := make([]float32, int(width)*int(height)*4)
	o := 0
	switch format {
	case formatR8Snorm:
		for i := 0; i < len(data); i++ {
			v := snorm8ToFloat(data[i])
			out[o] = v
			out[o+1] = v
			out[o+2] = v
			out[o+3] = 1.0
			o += 4
		}
	case formatRg8Snorm:
		for i := 0; i < len(data); i += 2 {
			out[o] = snorm8ToFloat(data[i])
			out[o+1] = snorm8ToFloat(data[i+1])
			out[o+2] = 0.0
			out[o+3] = 1.0
			o += 4
		}
	case formatRgba8Snorm:
		for i := 0; i < len(data); i += 4 {
			out[o] = snorm8ToFloat(data[i])
			out[o+1] = snorm8ToFloat(data[i+1])
			out[o+2] = snorm8ToFloat(data[i+2])
			out[o+3] = snorm8ToFloat(data[i+3])
			o += 4
		}
	case formatRf16:
		for i := 0; i < len(data); i += 2 {
			v := halfToFloat32(binary.LittleEndian.Uint16(data[i:]))
			out[o] = v
			out[o+1] = v
			out[o+2] = v
			out[o+3] = 1.0
			o += 4
		}
	case formatRgf16:
		for i := 0; i < len(data); i += 4 {
			out[o] = halfToFloat32(binary.LittleEndian.Uint16(data[i:]))
			out[o+1] = halfToFloat32(binary.LittleEndian.Uint16(data[i+2:]))
			out[o+2] = 0.0
			out[o+3] = 1.0
			o += 4
		}
	case formatRgbaf16:
		for i := 0; i < len(data); i += 8 {
			out[o] = halfToFloat32(binary.LittleEndian.Uint16(data[i:]))
			out[o+1] = halfToFloat32(binary.LittleEndian.Uint16(data[i+2:]))
			out[o+2] = halfToFloat32(binary.LittleEndian.Uint16(data[i+4:]))
			out[o+3] = halfToFloat32(binary.LittleEndian.Uint16(data[i+6:]))
			o += 4
		}
	case formatRf32:
		for i := 0; i < len(data); i += 4 {
			v := mathFloat32frombits(binary.LittleEndian.Uint32(data[i:]))
			out[o] = v
			out[o+1] = v
			out[o+2] = v
			out[o+3] = 1.0
			o += 4
		}
	case formatRgf32:
		for i := 0; i < len(data); i += 8 {
			out[o] = mathFloat32frombits(binary.LittleEndian.Uint32(data[i:]))
			out[o+1] = mathFloat32frombits(binary.LittleEndian.Uint32(data[i+4:]))
			out[o+2] = 0.0
			out[o+3] = 1.0
			o += 4
		}
	case formatRgbf32:
		for i := 0; i < len(data); i += 12 {
			out[o] = mathFloat32frombits(binary.LittleEndian.Uint32(data[i:]))
			out[o+1] = mathFloat32frombits(binary.LittleEndian.Uint32(data[i+4:]))
			out[o+2] = mathFloat32frombits(binary.LittleEndian.Uint32(data[i+8:]))
			out[o+3] = 1.0
			o += 4
		}
	case formatRgbaf32:
		for i := 0; i < len(data); i += 16 {
			out[o] = mathFloat32frombits(binary.LittleEndian.Uint32(data[i:]))
			out[o+1] = mathFloat32frombits(binary.LittleEndian.Uint32(data[i+4:]))
			out[o+2] = mathFloat32frombits(binary.LittleEndian.Uint32(data[i+8:]))
			out[o+3] = mathFloat32frombits(binary.LittleEndian.Uint32(data[i+12:]))
			o += 4
		}
	case formatR16:
		for i := 0; i < len(data); i += 2 {
			v := float32(binary.LittleEndian.Uint16(data[i:])) / 65535.0
			out[o] = v
			out[o+1] = v
			out[o+2] = v
			out[o+3] = 1.0
			o += 4
		}
	case formatR16Snorm:
		for i := 0; i < len(data); i += 2 {
			v := snorm16ToFloat(binary.LittleEndian.Uint16(data[i:]))
			out[o] = v
			out[o+1] = v
			out[o+2] = v
			out[o+3] = 1.0
			o += 4
		}
	case formatRg16:
		for i := 0; i < len(data); i += 4 {
			out[o] = float32(binary.LittleEndian.Uint16(data[i:])) / 65535.0
			out[o+1] = float32(binary.LittleEndian.Uint16(data[i+2:])) / 65535.0
			out[o+2] = 0.0
			out[o+3] = 1.0
			o += 4
		}
	case formatRg16Snorm:
		for i := 0; i < len(data); i += 4 {
			out[o] = snorm16ToFloat(binary.LittleEndian.Uint16(data[i:]))
			out[o+1] = snorm16ToFloat(binary.LittleEndian.Uint16(data[i+2:]))
			out[o+2] = 0.0
			out[o+3] = 1.0
			o += 4
		}
	case formatRgba16:
		for i := 0; i < len(data); i += 8 {
			out[o] = float32(binary.LittleEndian.Uint16(data[i:])) / 65535.0
			out[o+1] = float32(binary.LittleEndian.Uint16(data[i+2:])) / 65535.0
			out[o+2] = float32(binary.LittleEndian.Uint16(data[i+4:])) / 65535.0
			out[o+3] = float32(binary.LittleEndian.Uint16(data[i+6:])) / 65535.0
			o += 4
		}
	case formatRgba16Snorm:
		for i := 0; i < len(data); i += 8 {
			out[o] = snorm16ToFloat(binary.LittleEndian.Uint16(data[i:]))
			out[o+1] = snorm16ToFloat(binary.LittleEndian.Uint16(data[i+2:]))
			out[o+2] = snorm16ToFloat(binary.LittleEndian.Uint16(data[i+4:]))
			out[o+3] = snorm16ToFloat(binary.LittleEndian.Uint16(data[i+6:]))
			o += 4
		}
	}
	return out, nil
}

// encodeRgbaU8 encodes RGBA8 pixels to the given format.
func encodeRgbaU8(format rgbaFormat, width, height uint32, data []uint8) ([]byte, error) {
	if _, err := validateLength(width, height, 4, len(data)); err != nil {
		return nil, err
	}
	count := int(width) * int(height)
	out := make([]byte, 0, count*format.pixelSize())
	buf := make([]byte, 16)
	for i := 0; i < count; i++ {
		r := data[i*4]
		g := data[i*4+1]
		b := data[i*4+2]
		a := data[i*4+3]
		switch format {
		case formatR8:
			out = append(out, r)
		case formatR8Snorm:
			out = append(out, unorm8ToSnorm8(r))
		case formatRg8:
			out = append(out, r, g)
		case formatRg8Snorm:
			out = append(out, unorm8ToSnorm8(r), unorm8ToSnorm8(g))
		case formatRgb8:
			out = append(out, r, g, b)
		case formatRgba8:
			out = append(out, r, g, b, a)
		case formatRgba8Snorm:
			out = append(out, unorm8ToSnorm8(r), unorm8ToSnorm8(g), unorm8ToSnorm8(b), unorm8ToSnorm8(a))
		case formatR16:
			binary.LittleEndian.PutUint16(buf, unorm8ToUnorm16(r))
			out = append(out, buf[:2]...)
		case formatR16Snorm:
			binary.LittleEndian.PutUint16(buf, uint16(unorm8ToSnorm16(r)))
			out = append(out, buf[:2]...)
		case formatRg16:
			binary.LittleEndian.PutUint16(buf, unorm8ToUnorm16(r))
			binary.LittleEndian.PutUint16(buf[2:], unorm8ToUnorm16(g))
			out = append(out, buf[:4]...)
		case formatRg16Snorm:
			binary.LittleEndian.PutUint16(buf, uint16(unorm8ToSnorm16(r)))
			binary.LittleEndian.PutUint16(buf[2:], uint16(unorm8ToSnorm16(g)))
			out = append(out, buf[:4]...)
		case formatRgba16:
			binary.LittleEndian.PutUint16(buf, unorm8ToUnorm16(r))
			binary.LittleEndian.PutUint16(buf[2:], unorm8ToUnorm16(g))
			binary.LittleEndian.PutUint16(buf[4:], unorm8ToUnorm16(b))
			binary.LittleEndian.PutUint16(buf[6:], unorm8ToUnorm16(a))
			out = append(out, buf[:8]...)
		case formatRgba16Snorm:
			binary.LittleEndian.PutUint16(buf, uint16(unorm8ToSnorm16(r)))
			binary.LittleEndian.PutUint16(buf[2:], uint16(unorm8ToSnorm16(g)))
			binary.LittleEndian.PutUint16(buf[4:], uint16(unorm8ToSnorm16(b)))
			binary.LittleEndian.PutUint16(buf[6:], uint16(unorm8ToSnorm16(a)))
			out = append(out, buf[:8]...)
		case formatRf16:
			binary.LittleEndian.PutUint16(buf, uint16(halfFromFloat32(float32(r)/255.0)))
			out = append(out, buf[:2]...)
		case formatRgf16:
			binary.LittleEndian.PutUint16(buf, uint16(halfFromFloat32(float32(r)/255.0)))
			binary.LittleEndian.PutUint16(buf[2:], uint16(halfFromFloat32(float32(g)/255.0)))
			out = append(out, buf[:4]...)
		case formatRgbaf16:
			binary.LittleEndian.PutUint16(buf, uint16(halfFromFloat32(float32(r)/255.0)))
			binary.LittleEndian.PutUint16(buf[2:], uint16(halfFromFloat32(float32(g)/255.0)))
			binary.LittleEndian.PutUint16(buf[4:], uint16(halfFromFloat32(float32(b)/255.0)))
			binary.LittleEndian.PutUint16(buf[6:], uint16(halfFromFloat32(float32(a)/255.0)))
			out = append(out, buf[:8]...)
		case formatRf32:
			binary.LittleEndian.PutUint32(buf, mathFloat32bits(float32(r)/255.0))
			out = append(out, buf[:4]...)
		case formatRgf32:
			binary.LittleEndian.PutUint32(buf, mathFloat32bits(float32(r)/255.0))
			binary.LittleEndian.PutUint32(buf[4:], mathFloat32bits(float32(g)/255.0))
			out = append(out, buf[:8]...)
		case formatRgbf32:
			binary.LittleEndian.PutUint32(buf, mathFloat32bits(float32(r)/255.0))
			binary.LittleEndian.PutUint32(buf[4:], mathFloat32bits(float32(g)/255.0))
			binary.LittleEndian.PutUint32(buf[8:], mathFloat32bits(float32(b)/255.0))
			out = append(out, buf[:12]...)
		case formatRgbaf32:
			binary.LittleEndian.PutUint32(buf, mathFloat32bits(float32(r)/255.0))
			binary.LittleEndian.PutUint32(buf[4:], mathFloat32bits(float32(g)/255.0))
			binary.LittleEndian.PutUint32(buf[8:], mathFloat32bits(float32(b)/255.0))
			binary.LittleEndian.PutUint32(buf[12:], mathFloat32bits(float32(a)/255.0))
			out = append(out, buf[:16]...)
		case formatBgr8:
			out = append(out, b, g, r)
		case formatBgra8:
			out = append(out, b, g, r, a)
		case formatBgra4:
			// Pack each channel into 4 bits.
			// Most significant bit -> AAAARRRRGGGGBBBB -> least significant bit.
			out = append(out,
				(unorm8ToUnorm4(g)<<4)|unorm8ToUnorm4(b),
				(unorm8ToUnorm4(a)<<4)|unorm8ToUnorm4(r))
		case formatBgr5A1:
			// Pack RGB channels into 5 bits and alpha into 1 bit.
			r5 := unorm8ToUnorm5(r)
			g5 := unorm8ToUnorm5(g)
			b5 := unorm8ToUnorm5(b)
			var a1 uint8
			if a > 0 {
				a1 = 1
			}
			bytes := (uint16(r5) << 2) | rotateRight16(uint16(g5), 3) | (uint16(b5) << 8) | (uint16(a1) << 7)
			binary.BigEndian.PutUint16(buf, bytes)
			out = append(out, buf[:2]...)
		}
	}
	return out, nil
}

// encodeRgbaF32 encodes RGBAF32 pixels to the given format. Formats without
// float storage fall back to the u8 encoding.
func encodeRgbaF32(format rgbaFormat, width, height uint32, data []float32) ([]byte, error) {
	switch format {
	case formatR8Snorm, formatRg8Snorm, formatRgba8Snorm,
		formatRf16, formatRgf16, formatRgbaf16,
		formatRf32, formatRgf32, formatRgbf32, formatRgbaf32,
		formatR16, formatR16Snorm, formatRg16, formatRg16Snorm,
		formatRgba16, formatRgba16Snorm:
		// handled below
	default:
		rgba8 := make([]uint8, len(data))
		for i, f := range data {
			rgba8[i] = f32ToU8(f * 255.0)
		}
		return encodeRgbaU8(format, width, height, rgba8)
	}

	if _, err := validateLength(width, height, 4, len(data)); err != nil {
		return nil, err
	}
	count := int(width) * int(height)
	out := make([]byte, 0, count*format.pixelSize())
	buf := make([]byte, 16)
	for i := 0; i < count; i++ {
		r := data[i*4]
		g := data[i*4+1]
		b := data[i*4+2]
		a := data[i*4+3]
		switch format {
		case formatR8Snorm:
			out = append(out, uint8(floatToSnorm8(r)))
		case formatRg8Snorm:
			out = append(out, uint8(floatToSnorm8(r)), uint8(floatToSnorm8(g)))
		case formatRgba8Snorm:
			out = append(out, uint8(floatToSnorm8(r)), uint8(floatToSnorm8(g)), uint8(floatToSnorm8(b)), uint8(floatToSnorm8(a)))
		case formatRf16:
			binary.LittleEndian.PutUint16(buf, uint16(halfFromFloat32(r)))
			out = append(out, buf[:2]...)
		case formatRgf16:
			binary.LittleEndian.PutUint16(buf, uint16(halfFromFloat32(r)))
			binary.LittleEndian.PutUint16(buf[2:], uint16(halfFromFloat32(g)))
			out = append(out, buf[:4]...)
		case formatRgbaf16:
			binary.LittleEndian.PutUint16(buf, uint16(halfFromFloat32(r)))
			binary.LittleEndian.PutUint16(buf[2:], uint16(halfFromFloat32(g)))
			binary.LittleEndian.PutUint16(buf[4:], uint16(halfFromFloat32(b)))
			binary.LittleEndian.PutUint16(buf[6:], uint16(halfFromFloat32(a)))
			out = append(out, buf[:8]...)
		case formatRf32:
			binary.LittleEndian.PutUint32(buf, mathFloat32bits(r))
			out = append(out, buf[:4]...)
		case formatRgf32:
			binary.LittleEndian.PutUint32(buf, mathFloat32bits(r))
			binary.LittleEndian.PutUint32(buf[4:], mathFloat32bits(g))
			out = append(out, buf[:8]...)
		case formatRgbf32:
			binary.LittleEndian.PutUint32(buf, mathFloat32bits(r))
			binary.LittleEndian.PutUint32(buf[4:], mathFloat32bits(g))
			binary.LittleEndian.PutUint32(buf[8:], mathFloat32bits(b))
			out = append(out, buf[:12]...)
		case formatRgbaf32:
			binary.LittleEndian.PutUint32(buf, mathFloat32bits(r))
			binary.LittleEndian.PutUint32(buf[4:], mathFloat32bits(g))
			binary.LittleEndian.PutUint32(buf[8:], mathFloat32bits(b))
			binary.LittleEndian.PutUint32(buf[12:], mathFloat32bits(a))
			out = append(out, buf[:16]...)
		case formatR16:
			binary.LittleEndian.PutUint16(buf, f32ToU16(r*65535.0))
			out = append(out, buf[:2]...)
		case formatR16Snorm:
			binary.LittleEndian.PutUint16(buf, uint16(floatToSnorm16(r)))
			out = append(out, buf[:2]...)
		case formatRg16:
			binary.LittleEndian.PutUint16(buf, f32ToU16(r*65535.0))
			binary.LittleEndian.PutUint16(buf[2:], f32ToU16(g*65535.0))
			out = append(out, buf[:4]...)
		case formatRg16Snorm:
			binary.LittleEndian.PutUint16(buf, uint16(floatToSnorm16(r)))
			binary.LittleEndian.PutUint16(buf[2:], uint16(floatToSnorm16(g)))
			out = append(out, buf[:4]...)
		case formatRgba16:
			binary.LittleEndian.PutUint16(buf, f32ToU16(r*65535.0))
			binary.LittleEndian.PutUint16(buf[2:], f32ToU16(g*65535.0))
			binary.LittleEndian.PutUint16(buf[4:], f32ToU16(b*65535.0))
			binary.LittleEndian.PutUint16(buf[6:], f32ToU16(a*65535.0))
			out = append(out, buf[:8]...)
		case formatRgba16Snorm:
			binary.LittleEndian.PutUint16(buf, uint16(floatToSnorm16(r)))
			binary.LittleEndian.PutUint16(buf[2:], uint16(floatToSnorm16(g)))
			binary.LittleEndian.PutUint16(buf[4:], uint16(floatToSnorm16(b)))
			binary.LittleEndian.PutUint16(buf[6:], uint16(floatToSnorm16(a)))
			out = append(out, buf[:8]...)
		}
	}
	return out, nil
}

func mathFloat32frombits(b uint32) float32 {
	return math.Float32frombits(b)
}

func mathFloat32bits(f float32) uint32 {
	return math.Float32bits(f)
}

func rotateLeft16(v uint16, n uint) uint16 {
	return v<<n | v>>(16-n)
}

func rotateRight16(v uint16, n uint) uint16 {
	return v>>n | v<<(16-n)
}
