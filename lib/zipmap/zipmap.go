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

package zipmap

import (
	"archive/zip"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type ZipMap struct {
	zips map[string]*zip.ReadCloser
}

func New() *ZipMap {
	rv := &ZipMap{}
	rv.zips = make(map[string]*zip.ReadCloser)
	return rv
}

func (z *ZipMap) Exists(filename string) bool {
	f, err := z.Get(filename)
	if err == nil {
		f.Close()
		return true
	}
	return false
}

func (z *ZipMap) Get(filename string) (io.ReadCloser, error) {
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
	elems := strings.Split(filename, "/")
	for i, elem := range elems {
		if len(elem) < 5 || elem[len(elem)-4:] != ".zip" {
			continue
		}
		zipfile := filepath.Join(elems[0 : i+1]...)
		read, ok := z.zips[zipfile]
		if !ok {
			// log.Printf("Opening zip file %s...\n", zipfile)
			read, err = zip.OpenReader(zipfile)
			if err != nil {
				log.Print(err)
				read = nil
			}
			z.zips[zipfile] = read
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

		return rv, os.ErrNotExist
	}

	return rv, os.ErrNotExist
}

func (z *ZipMap) CopyTo(filename, destination string) error {
	f, err := z.Get(filename)
	defer f.Close()

	if err != nil {
		return err
	} else {
		g, err := os.Create(destination)
		defer g.Close()
		if err != nil {
			return err
		}

		_, err = io.Copy(g, f)
		if err != nil {
			return err
		}
	}
	return nil
}
