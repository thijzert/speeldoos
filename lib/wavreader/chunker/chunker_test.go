package chunker

import (
	"testing"

	"io"
	"io/ioutil"
	"time"
)

type dummyTime struct {
	T     time.Time
	Slept bool
}

func (d *dummyTime) Now() time.Time {
	return d.T
}

func (d *dummyTime) Sleep() {
	d.Slept = true
	d.T = d.T.Add(1 * time.Millisecond)
}

func getInputSignal(n, l int, now time.Time) *chunkContainer {
	buf := make([]byte, l*n)
	for i := 0; i < n; i++ {
		for j := 0; j < l; j++ {
			buf[l*i+j] = byte(i)
		}
	}

	m := &chunkContainer{
		chunks:     make([]chunk, n+2),
		start:      0,
		end:        n,
		errorState: io.EOF,
	}
	for i := 0; i < n; i++ {
		m.chunks[i].contents = buf[l*i : l*(i+1)]
		m.chunks[i].embargo = now.Add(time.Duration(int64(i-n)) * time.Second)
		m.chunks[i].seqno = uint32(i)
	}

	return m
}

func TestReadAll(t *testing.T) {
	now := time.Now()
	m := getInputSignal(60, 1, now)

	chm := &chunkReader{
		parent:  m,
		current: -1,
		seqno:   0xffffffff,
		timeSource: &dummyTime{
			T: now,
		},
	}

	s, err := ioutil.ReadAll(chm)
	if err != nil {
		t.Error(err)
	}

	correct := true
	for i, c := range s {
		if int(c) != i {
			correct = false
		}
	}
	if !correct {
		t.Logf("Result bytes: (%d) %x", len(s), s)
		t.Fail()
	}
}

func TestEmbargo(t *testing.T) {
	n := 60
	now := time.Now()
	m := getInputSignal(n, 1, now)

	wbuf := make([]byte, n*2)
	for i := 0; i < n; i++ {
		clock := &dummyTime{
			T: now.Add(time.Duration(int64(i-n))*time.Second - 5*time.Millisecond),
		}

		chm := &chunkReader{
			parent:     m,
			current:    -1,
			seqno:      0xffffffff,
			timeSource: clock,
		}

		for j := 0; j < i; j++ {
			_, err := chm.Read(wbuf)
			if err != nil {
				t.Error(err)
			} else if clock.Slept {
				t.Logf("T+%d: Slept after %d reads", i, j)
				t.Fail()
				break
			}
		}

		_, err := chm.Read(wbuf)
		if err != nil {
			t.Error(err)
		} else if !clock.Slept {
			t.Logf("T+%d: still reading without sleeping", i)
			t.Fail()
		}
	}
}
