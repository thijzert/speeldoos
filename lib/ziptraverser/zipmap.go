// Copyright 2016 Thijs van Dijk. All rights reserved.
// Use of this source code is governed by the BSD 3-clause
// license that can be found in the LICENSE file.

/*
	Package zipmap provides a transparent way of opening files by path name,
	where some of the directory names are actually zip archive files.

	Usage:

	Create an empty zipmap. Afterwards, ask it to open a file.
		zm := zipmap.New()
		f, err := zm.Get("path/to/foo.zip/bar.txt")
		if err != nil {
			panic(err)
		}
		io.Copy(os.Stdout, f)

	A Zipmap will cache open file pointers to zip archives it's encountered.
	If you call `Get`, zipmap expects you to call Close() on the resulting
	ReadCloser before calling `Get` again.
	This library is not thread-safe in any way.
*/

package ziptraverser

import (
	"archive/zip"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

type zipMap struct {
	zipLRU  []mapElement
	maxSize int
}

type mapElement struct {
	Filename string
	Zip      *zip.ReadCloser
}

func newMap(size int) *zipMap {
	rv := &zipMap{
		// A list of open files. The most recently used zip reader is at index 0
		// of this list; the last one is at the end.
		zipLRU: make([]mapElement, 0, size),
		// The maximum size zipLRU can be. If len(zipLRU) is equal to maxSize and
		// a new file is opened, the zip reader at index 0 is closed
		maxSize: size,
	}

	return rv
}

func (z *zipMap) getZipHandle(zipfile string) (*zip.ReadCloser, error) {
	var read mapElement
	var i int
	ok := false
	for i, read = range z.zipLRU {
		if read.Filename == zipfile {
			ok = true
			break
		}
	}

	if ok {
		z.zipLRU = append(z.zipLRU[:i], z.zipLRU[i+1:]...)
	} else {
		if len(z.zipLRU) == z.maxSize {
			// We're at the maximum number of open files - close the least recently used one
			z.zipLRU[0].Zip.Close()
			copy(z.zipLRU, z.zipLRU[1:])
			z.zipLRU = z.zipLRU[:z.maxSize-1]
		}

		f, err := zip.OpenReader(zipfile)
		if err != nil {
			return nil, err
		}
		read = mapElement{zipfile, f}
	}

	z.zipLRU = append(z.zipLRU, read)

	return read.Zip, nil
}

func (z *zipMap) Exists(filename string) bool {
	f, err := z.Get(filename)
	if err == nil {
		f.Close()
		return true
	}
	return false
}

func (z *zipMap) Get(filename string) (io.ReadCloser, error) {
	rv, _ := os.Open(os.DevNull)

	var err error

	// Try opening the file itself, maybe that works...
	fi, err := os.Stat(filename)
	if err == nil {
		// Is it a regular file?
		if (fi.Mode() & os.ModeType) == 0 {
			return os.Open(filename)
		}
	}

	// FIXME: I'm of the opinion that this should work: elems := filepath.SplitList(filename)
	abs := ""
	elems := strings.Split(filename, "/")
	if elems[0] == "" {
		elems = elems[1:]
		abs = "/"
	}
	for i, elem := range elems {
		if len(elem) < 5 || elem[len(elem)-4:] != ".zip" {
			continue
		}
		zipfile := abs + filepath.Join(elems[0:i+1]...)
		read, err := z.getZipHandle(zipfile)
		if err != nil {
			log.Print(err)
		}

		if read == nil {
			continue
		}

		localfile := filepath.Join(elems[i+1:]...)

		for _, zfp := range read.File {
			if zfp.Name == localfile {
				return zfp.Open()
			}
		}

		return rv, errors.Wrapf(os.ErrNotExist, "file '%s' does not exist in '%s'", localfile, zipfile)
	}

	return rv, errors.Wrapf(os.ErrNotExist, "file '%s' does not exist", filename) // os.ErrNotExist
}

func (z *zipMap) CopyTo(filename, destination string) error {
	f, err := z.Get(filename)
	defer f.Close()

	if err != nil {
		return err
	}

	g, err := os.Create(destination)
	defer g.Close()
	if err != nil {
		return err
	}

	_, err = io.Copy(g, f)
	if err != nil {
		return err
	}

	return nil
}

func (z *zipMap) Close() {
	for _, m := range z.zipLRU {
		m.Zip.Close()
	}
	z.zipLRU = z.zipLRU[:0]
}
