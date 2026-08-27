package ddsutil

import "math"

// channel is the set of channel types used by pixel formats.
type channel interface {
	~uint8 | ~int8 | ~uint16 | ~int16 | ~float32
}

// half is a half-precision float stored as its 16-bit representation.
type half uint16

const halfZero = half(0)

// f32ToU8 saturating float-to-int cast (NaN becomes 0, clamped to 0..255).
func f32ToU8(f float32) uint8 {
	if f != f { // NaN
		return 0
	}
	if f <= 0.0 {
		return 0
	}
	if f >= 255.0 {
		return 255
	}
	return uint8(f)
}

// f32ToU16 saturating float-to-int cast (NaN becomes 0, clamped to 0..65535).
func f32ToU16(f float32) uint16 {
	if f != f {
		return 0
	}
	if f <= 0.0 {
		return 0
	}
	if f >= 65535.0 {
		return 65535
	}
	return uint16(f)
}

// toUnorm8 converts a channel value to an 8-bit unorm.
func toUnorm8[T channel](v T) uint8 {
	switch t := any(v).(type) {
	case uint8:
		return t
	case int8:
		return snorm8ToUnorm8(uint8(t))
	case uint16:
		return unorm16ToUnorm8(t)
	case int16:
		return snorm16ToUnorm8(uint16(t))
	case half:
		return f32ToU8(halfToFloat32(uint16(t)) * 255.0)
	case float32:
		return f32ToU8(t * 255.0)
	}
	panic("unreachable channel type")
}

// toF32 converts a channel value to a float32.
func toF32[T channel](v T) float32 {
	switch t := any(v).(type) {
	case uint8:
		return float32(t) / 255.0
	case int8:
		return snorm8ToFloat(uint8(t))
	case uint16:
		return float32(t) / 65535.0
	case int16:
		return snorm16ToFloat(uint16(t))
	case half:
		return halfToFloat32(uint16(t))
	case float32:
		return t
	}
	panic("unreachable channel type")
}

// fromUnorm8 converts an 8-bit unorm to a channel value.
func fromUnorm8[T channel](u uint8) T {
	switch any(*new(T)).(type) {
	case uint8:
		return any(u).(T)
	case int8:
		return any(int8(unorm8ToSnorm8(u))).(T)
	case uint16:
		return any(unorm8ToUnorm16(u)).(T)
	case int16:
		return any(unorm8ToSnorm16(u)).(T)
	case half:
		return any(halfFromFloat32(float32(u) / 255.0)).(T)
	case float32:
		return any(float32(u) / 255.0).(T)
	}
	panic("unreachable channel type")
}

// fromF32 converts a float32 to a channel value.
func fromF32[T channel](f float32) T {
	switch any(*new(T)).(type) {
	case uint8:
		return any(f32ToU8(f * 255.0)).(T)
	case int8:
		return any(floatToSnorm8(f)).(T)
	case uint16:
		return any(f32ToU16(f * 65535.0)).(T)
	case int16:
		return any(floatToSnorm16(f)).(T)
	case half:
		return any(halfFromFloat32(f)).(T)
	case float32:
		return any(f).(T)
	}
	panic("unreachable channel type")
}

func zeroOf[T channel]() T {
	var z T
	return z
}

// snorm8ToUnorm8 remaps [-1, 1] to [0, 1] to fit in an unsigned integer.
// Validated against decoding R8Snorm DDS with GPU and paint.net (DirectXTex).
func snorm8ToUnorm8(x uint8) uint8 {
	switch {
	case x < 128:
		return x + 128
	case x == 128:
		return 0
	default:
		return x - 129
	}
}

// unorm8ToSnorm8 is the inverse of snorm8ToUnorm8.
func unorm8ToSnorm8(x uint8) uint8 {
	if x >= 128 {
		return x - 128
	} else if x == 127 {
		return 0
	}
	return x + 129
}

// snorm8ToFloat converts a signed 8-bit value (bit pattern) to a float in [-1, 1].
func snorm8ToFloat(x uint8) float32 {
	v := float32(int8(x)) / 127.0
	if v < -1.0 {
		return -1.0
	}
	return v
}

// floatToSnorm8 converts a float in [-1, 1] to a signed 8-bit value.
func floatToSnorm8(x float32) int8 {
	if x < -1.0 {
		x = -1.0
	} else if x > 1.0 {
		x = 1.0
	}
	return int8(roundF32(x * 127.0))
}

// snorm16ToFloat converts a signed 16-bit value (bit pattern) to a float in [-1, 1].
func snorm16ToFloat(x uint16) float32 {
	v := float32(int16(x)) / 32767.0
	if v < -1.0 {
		return -1.0
	}
	return v
}

// floatToSnorm16 converts a float in [-1, 1] to a signed 16-bit value.
func floatToSnorm16(x float32) int16 {
	if x < -1.0 {
		x = -1.0
	} else if x > 1.0 {
		x = 1.0
	}
	return int16(roundF32(x * 32767.0))
}

// roundF32 rounds half away from zero.
func roundF32(x float32) float32 {
	if x != x || x == 0 {
		return x
	}
	if x >= 8388608.0 || x <= -8388608.0 {
		return x // magnitude beyond integer precision
	}
	if x > 0 {
		return float32(int32(x + 0.5))
	}
	return float32(int32(x - 0.5))
}

// unorm4ToUnorm8 expands a 4 bit channel to 8 bit.
// https://rundevelopment.github.io/blog/fast-unorm-conversions
func unorm4ToUnorm8(x uint8) uint8 {
	return x * 17
}

func unorm8ToUnorm4(x uint8) uint8 {
	return uint8((uint16(x)*15 + 135) >> 8)
}

func unorm16ToUnorm8(x uint16) uint8 {
	return uint8((uint32(x)*255 + 32895) >> 16)
}

func unorm8ToUnorm16(x uint8) uint16 {
	return uint16(x) * 257
}

func unorm5ToUnorm8(x uint8) uint8 {
	return uint8((uint16(x)*527 + 23) >> 6)
}

func unorm8ToUnorm5(x uint8) uint8 {
	return uint8((uint16(x)*249 + 1014) >> 11)
}

// snorm16ToUnorm8 remaps [-1, 1] to [0, 1] to fit in an unsigned integer.
func snorm16ToUnorm8(x uint16) uint8 {
	return f32ToU8(roundF32((snorm16ToFloat(x)*0.5 + 0.5) * 255.0))
}

// unorm8ToSnorm16 remaps [0, 1] to [-1, 1] to fit in a signed integer.
func unorm8ToSnorm16(x uint8) int16 {
	return int16(roundF32(((float32(x)/255.0)*2.0 - 1.0) * 32767.0))
}

// halfToFloat32 converts a half-precision float to a float32.
// Ported from bcdec's half_to_float_quick (modified half_to_float_fast4
// from https://gist.github.com/rygorous/2144712).
func halfToFloat32(halfBits uint16) float32 {
	magic := float32frombits(113 << 23)
	shiftedExp := uint32(0x7c00 << 13) // exponent mask after shift

	o := uint32(halfBits&0x7fff) << 13 // exponent/mantissa bits
	exp := shiftedExp & o              // just the exponent
	o += (127 - 15) << 23              // exponent adjust

	// handle exponent special cases
	if exp == shiftedExp {
		// Inf/NaN?
		o += (128 - 16) << 23 // extra exp adjust
		// NaN: keep the payload but set the most significant mantissa bit
		// (0x7FC00000 | man << 13).
		if halfBits&0x03FF != 0 {
			o |= 1 << 22
		}
	} else if exp == 0 {
		// Zero/Denormal?
		o += 1 << 23                                // extra exp adjust
		o = float32bits(float32frombits(o) - magic) // renormalize
	}

	o |= uint32(halfBits&0x8000) << 16 // sign bit
	return float32frombits(o)
}

// float32ToHalf converts a float32 to half-precision with round-to-nearest-even.
func float32ToHalf(f float32) uint16 {
	bits := float32bits(f)
	sign := uint16((bits >> 16) & 0x8000)
	exp := int32((bits>>23)&0xFF) - 127 + 15
	mant := bits & 0x7FFFFF

	if exp <= 0 {
		// Denormal or zero in half: flush to zero for simplicity of edge cases,
		// but attempt proper denormal encoding like f16::from_f32.
		if exp < -10 {
			return sign
		}
		// Mantissa shifted to fit in 10 bits as a denormal.
		mant |= 0x800000
		shift := uint32(14 - exp)
		halfMant := mant >> shift
		// Round to nearest even.
		rem := mant & ((1 << shift) - 1)
		halfway := uint32(1) << (shift - 1)
		if rem > halfway || (rem == halfway && halfMant&1 == 1) {
			halfMant++
		}
		return sign | uint16(halfMant)
	} else if exp >= 0x1F {
		// Inf or NaN (NaN keeps a nonzero mantissa).
		if mant != 0 {
			return sign | 0x7E00 | uint16(mant>>13)
		}
		return sign | 0x7C00
	}

	// Normal number: round mantissa to 10 bits with round-to-nearest-even.
	halfMant := mant >> 13
	rem := mant & 0x1FFF
	if rem > 0x1000 || (rem == 0x1000 && halfMant&1 == 1) {
		halfMant++
		if halfMant == 0x400 {
			halfMant = 0
			exp++
			if exp >= 0x1F {
				return sign | 0x7C00
			}
		}
	}
	return sign | uint16(exp)<<10 | uint16(halfMant)
}

// halfFromFloat32 converts a float32 to a half value (round to nearest even).
func halfFromFloat32(f float32) half {
	return half(float32ToHalf(f))
}

func float32frombits(b uint32) float32 {
	return math.Float32frombits(b)
}

func float32bits(f float32) uint32 {
	return math.Float32bits(f)
}
