package ddsutil

// The MIT License (MIT)
//
// Copyright (c) 2018 Michael Dilger
//
// ... (see LICENSE for the full text)

import (
	"errors"
	"fmt"
	"io"
)

// Sentinel errors returned by the container layer. Use errors.Is to test
// for them; returned errors may wrap them with more detail.
var (
	// ErrBadMagicNumber is returned when the file does not start with "DDS ".
	ErrBadMagicNumber = errors.New("bad magic number")
	// ErrShortFile is returned when the file/stream is cut short.
	ErrShortFile = errors.New("file is cut short")
	// ErrUnsupportedFormat is returned when the format is not supported well
	// enough for the requested operation.
	ErrUnsupportedFormat = errors.New("format is not supported well enough for this operation")
	// ErrOutOfBounds is returned when a request is out of bounds.
	ErrOutOfBounds = errors.New("request is out of bounds")
	// ErrTooLarge is returned when a texture size computation overflows.
	ErrTooLarge = errors.New("texture size overflows the supported range")
	// ErrInvalidField is wrapped by errors reporting an invalid header field
	// or argument.
	ErrInvalidField = errors.New("invalid field")
	// ErrGeneral is wrapped by errors that fit no other category.
	ErrGeneral = errors.New("general error")
)

// GeneralError returns an error wrapping ErrGeneral with the given message.
func GeneralError(msg string) error {
	return fmt.Errorf("%w: %s", ErrGeneral, msg)
}

// InvalidFieldError returns an error wrapping ErrInvalidField that names the
// invalid field.
func InvalidFieldError(field string) error {
	return fmt.Errorf("%w: %s", ErrInvalidField, field)
}

// mapIO converts unexpected-end-of-stream errors to ErrShortFile and passes
// everything else through unchanged.
func mapIO(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		return ErrShortFile
	}
	return err
}
