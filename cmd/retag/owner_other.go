//go:build !unix

package main

import "os"

// copyOwner does nothing on platforms without Unix file ownership.
func copyOwner(*os.File, os.FileInfo) {}
