// Copyright 2016-2020 Thijs van Dijk. All rights reserved.
// Use of this source code is governed by the BSD 3-clause
// license that can be found in the LICENSE file.

/*
Package ziptraverser provides a transparent way of opening files by path
name, where some of the directory names are actually zip archive files.

Usage:

Create an empty ziptraverser. Afterwards, ask it to open a file.
	zm := ziptraverser.New()
	f, err := zm.Get("path/to/foo.zip/bar.txt")
	if err != nil {
		panic(err)
	}
	io.Copy(os.Stdout, f)

A Ziptraverser will cache open file pointers to zip archives it's
encountered. If you call `Get`, ziptraverser expects you to call Close()
on the resulting ReadCloser before calling `Get` again. This library is
not thread-safe in any way.
*/
package ziptraverser

import (
	"io"
)

// A ZipTraverser provides a transparent way of opening files by path name,
// where some of the directory names are actually zip archive files.
type ZipTraverser interface {
	// Exists tests if the given filename exists
	Exists(filename string) bool

	// Get opens the specified file name and provides a reader into its contents
	Get(filename string) (io.ReadCloser, error)

	// CopyTo copies the source file to a destination on the local file system
	CopyTo(filename, destination string) error

	// Close cleans up any open files in use
	Close()
}

// New instantiates a new ZipTraverser
func New() ZipTraverser {
	return newMap(2)
}
