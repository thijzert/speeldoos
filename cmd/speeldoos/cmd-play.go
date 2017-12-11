package main

import (
	"encoding/binary"
	"log"
	"math"
	"os"
	"os/exec"
)

func play_main(args []string) {
	// 3 seconds of 207.5 Hz
	buf := make([]byte, 3*48000*4)
	for i := 0; i < 3*48000; i++ {
		var t float64 = float64(i) / 48000.0
		y := int(math.Sin(415.0*t*math.Pi) * 10000.0)
		binary.LittleEndian.PutUint16(buf[4*i:], uint16(y))
		binary.LittleEndian.PutUint16(buf[4*i+2:], uint16(y))
	}

	var n, i int

	mpl := exec.Command(Config.Tools.MPlayer, "-noconsolecontrols", "-cache", "1024", "-rawaudio", "rate=48000:channels=2:samplesize=2", "-demuxer", "rawaudio", "-")

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
		n += i
		if n >= len(buf) {
			break
		}
	}

	output.Close()
}
