package chunker

import (
	"testing"

	"bytes"
	"io"
	"io/ioutil"
	"time"
)

func TestReadAll(t *testing.T) {
	n := 60
	l := 1
	buf := make([]byte, l*n)
	for i := 0; i < n; i++ {
		for j := 0; j < l; j++ {
			buf[l*i+j] = byte(i)
		}
	}

	t.Logf("Input bytes:  (%d) %x", len(buf), buf)

	emb := time.Now().Add(-10 * time.Millisecond)

	m := &chunkContainer{
		chunks:     make([]chunk, n+2),
		start:      0,
		end:        n,
		errorState: io.EOF,
	}
	for i := 0; i < n; i++ {
		m.chunks[i].contents = buf[l*i : l*(i+1)]
		m.chunks[i].embargo = emb
		m.chunks[i].seqno = uint32(i)
	}

	chm := &chunkReader{
		parent:  m,
		current: -1,
		seqno:   0xffffffff,
	}

	s, err := ioutil.ReadAll(chm)
	if err != nil {
		t.Error(err)
	} else if !bytes.Equal(s, buf) {
		t.Logf("Result bytes: (%d) %x", len(s), s)
		t.Fail()
	}
}
