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

// Sentinel errors returned by the container layer.
var (
	// ErrBadMagicNumber is returned when the file does not start with "DDS ".
	ErrBadMagicNumber = errors.New("bad magic number")
	// ErrShortFile is returned when the file/stream is cut short.
	ErrShortFile = errors.New("file is cut short")
	// ErrUnsupportedFormat is returned when the format is not supported well
	// enough for the requested operation.
	ErrUnsupportedFormat = errors.New("Format is not supported well enough for this operation")
	// ErrOutOfBounds is returned when a request is out of bounds.
	ErrOutOfBounds = errors.New("request is out of bounds")
)

// GeneralError returns a generic error carrying the given message.
func GeneralError(msg string) error {
	return fmt.Errorf("general error: %s", msg)
}

// InvalidFieldError returns an error reporting the named invalid field.
func InvalidFieldError(field string) error {
	return fmt.Errorf("invalid field: %s", field)
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
