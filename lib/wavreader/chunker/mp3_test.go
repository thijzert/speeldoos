package chunker

import (
	"crypto/sha512"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

type toms struct {
	f   *os.File
	key [64]byte
	pos int
}

func tomsDiner() (io.ReadCloser, error) {
	key, err := ioutil.ReadFile("testdata/license.txt")
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("No license could be found. To continue, please reach out to Suzanne Vega and ask for her permission to use a rendition of \"Tom's Diner\" to run \"go test.\"")
		} else {
			return nil, err
		}
	}
	f, err := os.Open("testdata/toms-diner.bin")
	if err != nil {
		return nil, err
	}

	rv := &toms{
		f:   f,
		key: sha512.Sum512(key),
		pos: 0,
	}
	return rv, nil
}

func (t *toms) Read(buf []byte) (int, error) {
	n, err := t.f.Read(buf)
	for i, c := range buf[:n] {
		buf[i] = c ^ t.key[(i+t.pos)%64]
	}
	t.pos += n
	return n, err
}

func (t *toms) Close() error {
	return t.f.Close()
}

func TestTomsDiner(t *testing.T) {
	in, err := tomsDiner()
	if err != nil {
		t.Error(err)
		return
	}
	defer in.Close()

	out, err := os.Create("testdata/toms-diner.mp3")
	if err != nil {
		t.Error(err)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		t.Error(err)
		return
	}
}
