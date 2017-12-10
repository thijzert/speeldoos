package main

import (
	"encoding/binary"
	"fmt"
	wav "github.com/thijzert/speeldoos/lib/wavreader"
	"log"
	"math"
	"os"
	"os/exec"
)

func main() {
	fmt.Println("vim-go")

	// 3 seconds of 207.5 Hz
	buf := make([]byte, 3*48000*4)
	for i := 0; i < 3*48000; i++ {
		var t float64 = float64(i) / 48000.0
		y := int(math.Sin(415.0*t*math.Pi) * 10000.0)
		binary.LittleEndian.PutUint16(buf[4*i:], uint16(y))
		binary.LittleEndian.PutUint16(buf[4*i+2:], uint16(y))
	}

	var n, i int

	of, err := os.Create("out.wav")
	ww := wav.NewWriter(of, 1, 2, 48000, 16)
	if err != nil {
		log.Fatal(err)
	}

	for err == nil {
		i, err = ww.Write(buf[n:])
		n += i
		if n >= len(buf) {
			break
		}
	}
	err = ww.Close()

	if true {
		mpl := exec.Command("mplayer", "-noconsolecontrols", "-cache", "1024", "-rawaudio", "rate=48000:channels=2:samplesize=2", "-demuxer", "rawaudio", "-")
		//mpl := exec.Command("mplayer", "-cache", "1024", "-")

		mpl.Stdout = os.Stdout
		mpl.Stderr = os.Stderr

		output, err := mpl.StdinPipe()
		if err != nil {
			log.Fatal(err)
		}

		mpl.Start()
		defer mpl.Wait()

		i = 0
		n = 0
		for err == nil {
			i, err = output.Write(buf[n:])
			fmt.Printf("wrote %d bytes\n", i)
			n += i
			if n >= len(buf) {
				break
			}
		}

		output.Close()
	}
}
