package main

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/thijzert/speeldoos/lib/hivemind"
)

type waiter int64

func (w waiter) Run(h hivemind.JC) error {
	b := make([]byte, 2)
	rand.Reader.Read(b)

	h.SetTitle(fmt.Sprintf("%02x", b))

	if b[0]&0x7 == 0 {
		h.Printf("Sleeping %d ms...", w)
	}
	if b[0]&0x3f == 0 {
		h.Printf("%s", `
                   __
                  // \
                  \\_/ //
''-.._.-''-.._.. -(||)(')
                  '''`)
	}

	time.Sleep(time.Duration(int64(w)) * time.Millisecond)

	if b[0]&0x7 == 0 {
		h.Println("Done sleeping")
	}
	if b[1]&0xf == 0 {
		return fmt.Errorf("Simulated error %2x", b[0])
	}
	return nil
}

func main() {
	h := hivemind.New(0)

	b := make([]byte, 1)
	for i := 0; i < 30; i++ {
		rand.Reader.Read(b)
		j := waiter(300 + 10*int64(b[0]))
		h.AddJob(j)
	}

	h.Wait()
}
