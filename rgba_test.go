package ddsutil

import (
	"encoding/binary"
	"math"
	"testing"
)

func f32s(vals ...float32) []float32 { return vals }

func u8s(vals ...uint8) []uint8 { return vals }

func TestR8FromRgba8(t *testing.T) {
	got, err := encodeRgbaU8(formatR8, 1, 1, u8s(1, 2, 3, 4))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(1)) {
		t.Errorf("got %v", got)
	}
}

func TestRgba8FromR8(t *testing.T) {
	got, err := decodeRgbaU8(formatR8, 1, 1, u8s(64))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(64, 64, 64, 255)) {
		t.Errorf("got %v", got)
	}
}

func TestR8SnormFromRgba8(t *testing.T) {
	got, err := encodeRgbaU8(formatR8Snorm, 1, 1, u8s(1, 2, 3, 4))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(130)) {
		t.Errorf("got %v", got)
	}
}

func TestR8SnormFromRgbaf32(t *testing.T) {
	got, err := encodeRgbaF32(formatR8Snorm, 1, 1, f32s(-1.0, 0.0, 1.0, 1.0))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(129)) {
		t.Errorf("got %v", got)
	}
}

func TestRgba8FromR8Snorm(t *testing.T) {
	got, err := decodeRgbaU8(formatR8Snorm, 1, 1, u8s(64))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(192, 192, 192, 255)) {
		t.Errorf("got %v", got)
	}
}

func TestRgbaf32FromR8Snorm(t *testing.T) {
	got, err := decodeRgbaF32(formatR8Snorm, 1, 1, u8s(128))
	if err != nil {
		t.Fatal(err)
	}
	if !equalF32(got, f32s(-1.0, -1.0, -1.0, 1.0)) {
		t.Errorf("got %v", got)
	}
}

func TestRg8FromRgba8(t *testing.T) {
	got, err := encodeRgbaU8(formatRg8, 1, 1, u8s(1, 2, 3, 4))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(1, 2)) {
		t.Errorf("got %v", got)
	}
}

func TestRg8SnormFromRgba8(t *testing.T) {
	got, err := encodeRgbaU8(formatRg8Snorm, 1, 1, u8s(1, 2, 3, 4))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(130, 131)) {
		t.Errorf("got %v", got)
	}
}

func TestRg8SnormFromRgbaf32(t *testing.T) {
	got, err := encodeRgbaF32(formatRg8Snorm, 1, 1, f32s(-1.0, 0.0, 1.0, 1.0))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(129, 0)) {
		t.Errorf("got %v", got)
	}
}

func TestRgba8FromRg8Snorm(t *testing.T) {
	got, err := decodeRgbaU8(formatRg8Snorm, 1, 1, u8s(1, 2))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(129, 130, 128, 255)) {
		t.Errorf("got %v", got)
	}
}

func TestRgbaf32FromRg8Snorm(t *testing.T) {
	got, err := decodeRgbaF32(formatRg8Snorm, 1, 1, u8s(128, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !equalF32(got, f32s(-1.0, 0.0, 0.0, 1.0)) {
		t.Errorf("got %v", got)
	}
}

func TestRgba8SnormFromRgba8(t *testing.T) {
	got, err := encodeRgbaU8(formatRgba8Snorm, 1, 1, u8s(1, 2, 3, 4))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(130, 131, 132, 133)) {
		t.Errorf("got %v", got)
	}
}

func TestRgba8SnormFromRgbaf32(t *testing.T) {
	got, err := encodeRgbaF32(formatRgba8Snorm, 1, 1, f32s(-1.0, 0.0, 1.0, 1.0))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(129, 0, 127, 127)) {
		t.Errorf("got %v", got)
	}
}

func TestRgba8FromRgba8Snorm(t *testing.T) {
	got, err := decodeRgbaU8(formatRgba8Snorm, 1, 1, u8s(1, 2, 3, 4))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(129, 130, 131, 132)) {
		t.Errorf("got %v", got)
	}
}

func TestRgbaf32FromRgba8Snorm(t *testing.T) {
	got, err := decodeRgbaF32(formatRgba8Snorm, 1, 1, u8s(128, 0, 129, 127))
	if err != nil {
		t.Fatal(err)
	}
	if !equalF32(got, f32s(-1.0, 0.0, -1.0, 1.0)) {
		t.Errorf("got %v", got)
	}
}

func TestBgra8FromRgba8(t *testing.T) {
	got, err := encodeRgbaU8(formatBgra8, 1, 1, u8s(1, 2, 3, 4))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(3, 2, 1, 4)) {
		t.Errorf("got %v", got)
	}
}

func TestRgba8FromBgra8(t *testing.T) {
	got, err := decodeRgbaU8(formatBgra8, 1, 1, u8s(1, 2, 3, 4))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(3, 2, 1, 4)) {
		t.Errorf("got %v", got)
	}
}

func TestRgb8FromRgba8(t *testing.T) {
	got, err := encodeRgbaU8(formatRgb8, 1, 1, u8s(1, 2, 3, 4))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(1, 2, 3)) {
		t.Errorf("got %v", got)
	}
}

func TestRgba8FromBgr8(t *testing.T) {
	got, err := decodeRgbaU8(formatBgr8, 1, 1, u8s(1, 2, 3))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(3, 2, 1, 255)) {
		t.Errorf("got %v", got)
	}
}

func TestRgba8FromRgf32(t *testing.T) {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint32(data, math.Float32bits(0.0))
	binary.LittleEndian.PutUint32(data[4:], math.Float32bits(0.25))
	got, err := decodeRgbaU8(formatRgf32, 1, 1, data)
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(0, 63, 0, 255)) {
		t.Errorf("got %v", got)
	}
}

func TestRgf32FromRgba8(t *testing.T) {
	got, err := encodeRgbaU8(formatRgf32, 1, 1, u8s(0, 51, 153, 255))
	if err != nil {
		t.Fatal(err)
	}
	want := make([]byte, 8)
	binary.LittleEndian.PutUint32(want, math.Float32bits(0.0))
	binary.LittleEndian.PutUint32(want[4:], math.Float32bits(0.2))
	if !equalBytes(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestRgbaf32FromRgf32(t *testing.T) {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint32(data, math.Float32bits(0.0))
	binary.LittleEndian.PutUint32(data[4:], math.Float32bits(0.25))
	got, err := decodeRgbaF32(formatRgf32, 1, 1, data)
	if err != nil {
		t.Fatal(err)
	}
	if !equalF32(got, f32s(0.0, 0.25, 0.0, 1.0)) {
		t.Errorf("got %v", got)
	}
}

func TestRgba8FromRgbaf32(t *testing.T) {
	data := make([]byte, 16)
	binary.LittleEndian.PutUint32(data, math.Float32bits(0.0))
	binary.LittleEndian.PutUint32(data[4:], math.Float32bits(0.25))
	binary.LittleEndian.PutUint32(data[8:], math.Float32bits(0.5))
	binary.LittleEndian.PutUint32(data[12:], math.Float32bits(1.0))
	got, err := decodeRgbaU8(formatRgbaf32, 1, 1, data)
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(0, 63, 127, 255)) {
		t.Errorf("got %v", got)
	}
}

func TestRgbaf32FromRgba8(t *testing.T) {
	got, err := encodeRgbaU8(formatRgbaf32, 1, 1, u8s(0, 51, 153, 255))
	if err != nil {
		t.Fatal(err)
	}
	want := make([]byte, 16)
	binary.LittleEndian.PutUint32(want, math.Float32bits(0.0))
	binary.LittleEndian.PutUint32(want[4:], math.Float32bits(0.2))
	binary.LittleEndian.PutUint32(want[8:], math.Float32bits(0.6))
	binary.LittleEndian.PutUint32(want[12:], math.Float32bits(1.0))
	if !equalBytes(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestRgba8FromRgbaf16(t *testing.T) {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint16(data, float32ToHalf(0.0))
	binary.LittleEndian.PutUint16(data[2:], float32ToHalf(0.25))
	binary.LittleEndian.PutUint16(data[4:], float32ToHalf(0.5))
	binary.LittleEndian.PutUint16(data[6:], float32ToHalf(1.0))
	got, err := decodeRgbaU8(formatRgbaf16, 1, 1, data)
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(0, 63, 127, 255)) {
		t.Errorf("got %v", got)
	}
}

func TestRgbaf16FromRgba8(t *testing.T) {
	got, err := encodeRgbaU8(formatRgbaf16, 1, 1, u8s(0, 51, 153, 255))
	if err != nil {
		t.Fatal(err)
	}
	want := make([]byte, 8)
	binary.LittleEndian.PutUint16(want, float32ToHalf(0.0))
	binary.LittleEndian.PutUint16(want[2:], float32ToHalf(0.2))
	binary.LittleEndian.PutUint16(want[4:], float32ToHalf(0.6))
	binary.LittleEndian.PutUint16(want[6:], float32ToHalf(1.0))
	if !equalBytes(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestBgra4FromRgba8(t *testing.T) {
	got, err := encodeRgbaU8(formatBgra4, 1, 1, u8s(255, 51, 0, 204))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(0x30, 0xCF)) {
		t.Errorf("got %v", got)
	}
}

func TestRgba8FromBgra4(t *testing.T) {
	got, err := decodeRgbaU8(formatBgra4, 1, 1, u8s(0x30, 0xCF))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(255, 51, 0, 204)) {
		t.Errorf("got %v", got)
	}
}

func TestBgr5A1FromRgba8(t *testing.T) {
	got, err := encodeRgbaU8(formatBgr5A1, 2, 1, u8s(107, 74, 66, 255, 115, 74, 74, 255))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(0x28, 0xB5, 0x29, 0xB9)) {
		t.Errorf("got %v", got)
	}
}

func TestRgba8FromBgr5A1(t *testing.T) {
	got, err := decodeRgbaU8(formatBgr5A1, 2, 1, u8s(0x28, 0xB5, 0x29, 0xB9))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(107, 74, 66, 255, 115, 74, 74, 255)) {
		t.Errorf("got %v", got)
	}
}

func TestR16FromRgba8(t *testing.T) {
	got, err := encodeRgbaU8(formatR16, 1, 1, u8s(127, 128, 129, 130))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(127, 127)) {
		t.Errorf("got %v", got)
	}
}

func TestRgba8FromR16(t *testing.T) {
	got, err := decodeRgbaU8(formatR16, 1, 1, u8s(127, 127))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(127, 127, 127, 255)) {
		t.Errorf("got %v", got)
	}
}

func TestR16FromRgbaf32(t *testing.T) {
	got, err := encodeRgbaF32(formatR16, 1, 1, f32s(0.25, 0.75, 0.5, 1.0))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(255, 63)) {
		t.Errorf("got %v", got)
	}
}

func TestRgbaf32FromR16(t *testing.T) {
	got, err := decodeRgbaF32(formatR16, 1, 1, u8s(255, 255))
	if err != nil {
		t.Fatal(err)
	}
	if !equalF32(got, f32s(1.0, 1.0, 1.0, 1.0)) {
		t.Errorf("got %v", got)
	}
}

func TestR16SnormFromRgba8(t *testing.T) {
	got, err := encodeRgbaU8(formatR16Snorm, 1, 1, u8s(127, 128, 129, 130))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(128, 255)) {
		t.Errorf("got %v", got)
	}
}

func TestRgba8FromR16Snorm(t *testing.T) {
	got, err := decodeRgbaU8(formatR16Snorm, 1, 1, u8s(127, 127))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(255, 255, 255, 255)) {
		t.Errorf("got %v", got)
	}
}

func TestR16SnormFromRgbaf32(t *testing.T) {
	got, err := encodeRgbaF32(formatR16Snorm, 1, 1, f32s(-1.0, 0.0, 0.5, 1.0))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(1, 128)) {
		t.Errorf("got %v", got)
	}
}

func TestRgbaf32FromR16Snorm(t *testing.T) {
	got, err := decodeRgbaF32(formatR16Snorm, 1, 1, u8s(1, 128))
	if err != nil {
		t.Fatal(err)
	}
	if !equalF32(got, f32s(-1.0, -1.0, -1.0, 1.0)) {
		t.Errorf("got %v", got)
	}
}

func TestRgba16FromRgba8(t *testing.T) {
	got, err := encodeRgbaU8(formatRgba16, 1, 1, u8s(127, 128, 129, 130))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(127, 127, 128, 128, 129, 129, 130, 130)) {
		t.Errorf("got %v", got)
	}
}

func TestRgba8FromRgba16(t *testing.T) {
	got, err := decodeRgbaU8(formatRgba16, 1, 1, u8s(127, 127, 128, 128, 129, 129, 130, 130))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(127, 128, 129, 130)) {
		t.Errorf("got %v", got)
	}
}

func TestRgba16FromRgbaf32(t *testing.T) {
	got, err := encodeRgbaF32(formatRgba16, 1, 1, f32s(0.25, 0.75, 0.5, 1.0))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(255, 63, 255, 191, 255, 127, 255, 255)) {
		t.Errorf("got %v", got)
	}
}

func TestRgbaf32FromRgba16(t *testing.T) {
	got, err := decodeRgbaF32(formatRgba16, 1, 1, u8s(255, 255, 0, 0, 255, 255, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !equalF32(got, f32s(1.0, 0.0, 1.0, 0.0)) {
		t.Errorf("got %v", got)
	}
}

func TestRgba16SnormFromRgba8(t *testing.T) {
	got, err := encodeRgbaU8(formatRgba16Snorm, 1, 1, u8s(127, 128, 129, 130))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(128, 255, 128, 0, 129, 1, 130, 2)) {
		t.Errorf("got %v", got)
	}
}

func TestRgba8FromRgba16Snorm(t *testing.T) {
	got, err := decodeRgbaU8(formatRgba16Snorm, 1, 1, u8s(127, 127, 128, 128, 129, 129, 130, 130))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(255, 0, 1, 2)) {
		t.Errorf("got %v", got)
	}
}

func TestRgba16SnormFromRgbaf32(t *testing.T) {
	got, err := encodeRgbaF32(formatRgba16Snorm, 1, 1, f32s(-1.0, 0.0, 0.5, 1.0))
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(got, u8s(1, 128, 0, 0, 0, 64, 255, 127)) {
		t.Errorf("got %v", got)
	}
}

func TestRgbaf32FromRgba16Snorm(t *testing.T) {
	got, err := decodeRgbaF32(formatRgba16Snorm, 1, 1, u8s(1, 128, 0, 0, 0, 128, 255, 127))
	if err != nil {
		t.Fatal(err)
	}
	if !equalF32(got, f32s(-1.0, 0.0, -1.0, 1.0)) {
		t.Errorf("got %v", got)
	}
}

func TestConvertFunctions(t *testing.T) {
	// snorm8 -> unorm8 reference implementation.
	ref := func(x uint8) uint8 {
		return f32ToU8(roundF32((snorm8ToFloat(x)*0.5 + 0.5) * 255.0))
	}
	for i := 0; i <= 255; i++ {
		if got := snorm8ToUnorm8(uint8(i)); got != ref(uint8(i)) {
			t.Fatalf("snorm8ToUnorm8(%d) = %d want %d", i, got, ref(uint8(i)))
		}
	}

	// unorm8 -> snorm8 reference implementation.
	ref2 := func(x uint8) int8 {
		return int8(roundF32(((float32(x)/255.0)*2.0 - 1.0) * 127.0))
	}
	for i := 0; i <= 255; i++ {
		if got := int8(unorm8ToSnorm8(uint8(i))); got != ref2(uint8(i)) {
			t.Fatalf("unorm8ToSnorm8(%d) = %d want %d", i, got, ref2(uint8(i)))
		}
	}

	// unorm16 -> unorm8
	ref3 := func(x uint16) uint8 {
		return f32ToU8(roundF32(float32(x) / 65535.0 * 255.0))
	}
	for i := 0; i <= 65535; i += 7 {
		if got := unorm16ToUnorm8(uint16(i)); got != ref3(uint16(i)) {
			t.Fatalf("unorm16ToUnorm8(%d) = %d want %d", i, got, ref3(uint16(i)))
		}
	}
	// exhaustive unorm4/unorm5 checks
	ref4 := func(x uint8) uint8 { return f32ToU8(roundF32(float32(x) / 15.0 * 255.0)) }
	for i := 0; i <= 15; i++ {
		if got := unorm4ToUnorm8(uint8(i)); got != ref4(uint8(i)) {
			t.Fatalf("unorm4ToUnorm8(%d) = %d want %d", i, got, ref4(uint8(i)))
		}
	}
	ref5 := func(x uint8) uint8 { return f32ToU8(roundF32(float32(x) / 31.0 * 255.0)) }
	for i := 0; i <= 31; i++ {
		if got := unorm5ToUnorm8(uint8(i)); got != ref5(uint8(i)) {
			t.Fatalf("unorm5ToUnorm8(%d) = %d want %d", i, got, ref5(uint8(i)))
		}
	}
	// unorm8 -> unorm5
	ref6 := func(x uint8) uint8 { return f32ToU8(roundF32(float32(x) / 255.0 * 31.0)) }
	for i := 0; i <= 255; i++ {
		if got := unorm8ToUnorm5(uint8(i)); got != ref6(uint8(i)) {
			t.Fatalf("unorm8ToUnorm5(%d) = %d want %d", i, got, ref6(uint8(i)))
		}
	}
	// unorm8 -> unorm4
	ref7 := func(x uint8) uint8 { return f32ToU8(roundF32(float32(x) / 255.0 * 15.0)) }
	for i := 0; i <= 255; i++ {
		if got := unorm8ToUnorm4(uint8(i)); got != ref7(uint8(i)) {
			t.Fatalf("unorm8ToUnorm4(%d) = %d want %d", i, got, ref7(uint8(i)))
		}
	}
}

func TestHalfConversions(t *testing.T) {
	// Round trip through half with a few values.
	for _, f := range []float32{0.0, 1.0, 0.5, 0.25, 0.2, -1.0, 2.0, 65504.0, 0.1, 3.14159} {
		h := float32ToHalf(f)
		back := halfToFloat32(h)
		if math.Abs(float64(back-f)) > 0.001*math.Max(1, math.Abs(float64(f))) {
			t.Errorf("half round trip %v -> %v", f, back)
		}
	}
	// Known half bit patterns.
	if got := float32ToHalf(1.0); got != 0x3C00 {
		t.Errorf("float32ToHalf(1.0) = %#x want 0x3C00", got)
	}
	if got := float32ToHalf(0.5); got != 0x3800 {
		t.Errorf("float32ToHalf(0.5) = %#x want 0x3800", got)
	}
	if got := float32ToHalf(-2.0); got != 0xC000 {
		t.Errorf("float32ToHalf(-2.0) = %#x want 0xC000", got)
	}
	if got := halfToFloat32(0x3C00); got != 1.0 {
		t.Errorf("halfToFloat32(0x3C00) = %v want 1.0", got)
	}
}

func equalF32(a, b []float32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
