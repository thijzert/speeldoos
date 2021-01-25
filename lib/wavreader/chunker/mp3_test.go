package chunker

import (
	"crypto/sha512"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"
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
			return nil, fmt.Errorf("No license could be found. To continue, please reach out to Suzanne Vega and ask for her written permission to use a rendition of \"Tom's Diner\" to run \"go test.\"")
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

func TestMP3Splitting(t *testing.T) {
	td, err := tomsDiner()
	if err != nil {
		t.Error(err)
		return
	}

	pr, pw := io.Pipe()
	go func() {
		io.Copy(pw, td)
		pw.Close()
	}()

	tstart := time.Now().Add(-720 * 24 * time.Hour)

	rv := &mp3Chunker{
		mp3out:  pr,
		embargo: tstart,
		chcont: &chunkContainer{
			chunks: make([]chunk, 30),
			start:  0,
			end:    0,
		},
	}

	rv.splitChunks()

	dur := rv.embargo.Sub(tstart)
	if dur < (2*time.Minute+9*time.Second) || dur > (2*time.Minute+11*time.Second) {
		t.Errorf("Observed song duration: %s; should be: 2m10s", dur)
	}

	var stat Statuser = rv
	stat.BufferStatus()
}
