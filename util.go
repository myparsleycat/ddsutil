package ddsutil

// The MIT License (MIT)
//
// Copyright (c) 2018 Michael Dilger
//
// ... (see LICENSE for the full text)

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

// readU32 reads one little-endian uint32 from r. Short reads become
// ErrShortFile.
func readU32(r io.Reader) (uint32, error) {
	var buf [4]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, mapIO(err)
	}
	return binary.LittleEndian.Uint32(buf[:]), nil
}

// writeU32 writes one little-endian uint32 to w.
func writeU32(w io.Writer, v uint32) error {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	_, err := w.Write(buf[:])
	return mapIO(err)
}

// u32p returns a pointer to v; a small helper for optional (Option-style)
// fields.
func u32p(v uint32) *uint32 { return &v }

// u32Val dereferences p, or returns fallback when p is nil.
func u32Val(p *uint32, fallback uint32) uint32 {
	if p != nil {
		return *p
	}
	return fallback
}

// flagName pairs a bit with its display name for formatFlags.
type flagName[T ~uint32] struct {
	bit  T
	name string
}

// formatFlags renders a bitflag value as a "NAME1 | NAME2" string, appending
// any unknown bits in hex. Used by the String() methods of the flag types.
func formatFlags[T ~uint32](bits T, names []flagName[T]) string {
	var b strings.Builder
	rest := uint32(bits)
	for _, n := range names {
		bit := uint32(n.bit)
		if bit != 0 && rest&bit == bit {
			if b.Len() > 0 {
				b.WriteString(" | ")
			}
			b.WriteString(n.name)
			rest &^= bit
		}
	}
	if rest != 0 {
		if b.Len() > 0 {
			b.WriteString(" | ")
		}
		fmt.Fprintf(&b, "0x%x", rest)
	}
	if b.Len() == 0 {
		return "0x0"
	}
	return b.String()
}
