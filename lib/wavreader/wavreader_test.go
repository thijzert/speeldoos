package wavreader

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func TestWavHeader(t *testing.T) {
	// Happy flow: five channels, 65535Hz, 8 bit "hello"
	ww, err := parseWavString(t, "RIFF\x29\x00\x00\x00WAVEfmt \x10\x00\x00\x00\x01\x00\x05\x00\x00\x00\x01\x00\x00\x00\x05\x00\x05\x00\x08\x00data\x05\x00\x00\x00hello")
	if err != nil {
		t.Errorf("%v", err)
	} else {
		if ww.Format.Channels != 5 {
			t.Errorf("expected %d channels, got %d (%04x)", 5, ww.Format.Channels, ww.Format.Channels)
		}
		if ww.Format.Rate != 65536 {
			t.Errorf("expected %d Hz, got %d (%04x)", 65536, ww.Format.Rate, ww.Format.Rate)
		}
		if ww.Format.Bits != 8 {
			t.Errorf("expected %d bits, got %d (%04x)", 8, ww.Format.Bits, ww.Format.Bits)
		}
		if ww.Size != 5 {
			t.Errorf("expected %d bits, got %d (%04x)", 8, ww.Format.Bits, ww.Format.Bits)
		}

		buf := make([]byte, 5)
		n, err := io.ReadFull(ww, buf)
		if err != nil {
			t.Errorf("expeced %d bytes of data, got %d: %v", 5, n, err)
		}
		if string(buf) != "hello" {
			t.Errorf("expected \"hello\", got \"%s\" - %02x", buf, buf)
		}
		n, err = ww.Read(buf)

		if err != io.EOF {
			t.Errorf("expected end of file, instead got \"%s\" - %02x", buf, buf)
		}
	}

	_, err = parseWavString(t, "RAFF\x29\x00\x00\x00WAVEfmt \x10\x00\x00\x00\x01\x00\x05\x00\x00\x00\x01\x00\x00\x00\x05\x00\x05\x00\x08\x00data\x05\x00\x00\x00hello")
	if err == nil {
		t.Errorf("bogus 'raff' file should not have parsed")
	}

	_, err = parseWavString(t, "RIFF\x28\x00\x00\x00WAVEfmt \x0f\x00\x00\x00\x01\x00\x05\x00\x00\x00\x01\x00\x00\x00\x05\x00\x05\x00\x08\x00data\x05\x00\x00\x00hello")
	if err == nil {
		t.Errorf("far too small headers should not have parsed")
	}

	_, err = parseWavString(t, "RIFF\x29\x00\x00\x00W000fmt \x10\x00\x00\x00\x01\x00\x05\x00\x00\x00\x01\x00\x00\x00\x05\x00\x05\x00\x08\x00data\x05\x00\x00\x00hello")
	if err == nil {
		t.Errorf("bogus 'wooo' format should not have parsed")
	}

	_, err = parseWavString(t, "RIFF\x29\x00\x00\x00WAVEprtf\x10\x00\x00\x00\x01\x00\x05\x00\x00\x00\x01\x00\x00\x00\x05\x00\x05\x00\x08\x00data\x05\x00\x00\x00hello")
	if err == nil {
		t.Errorf("bogus 'printf' block should not have parsed")
	}

	_, err = parseWavString(t, "RIFF\x29\x00\x00\x00WAVEfmt \x10\x00\x00\x00\x01\x00\x05\x00\x00\x00\x01\x00\x01\x00\x05\x00\x05\x00\x08\x00data\x05\x00\x00\x00hello")
	if err == nil {
		t.Errorf("incorrect bytes per second should not have parsed")
	}

	_, err = parseWavString(t, "RIFF\x29\x00\x00\x00WAVEfmt \x10\x00\x00\x00\x01\x00\x05\x00\x00\x00\x01\x00\x00\x00\x05\x01\x05\x00\x08\x00data\x05\x00\x00\x00hello")
	if err == nil {
		t.Errorf("incorrect byte align should not have parsed")
	}

	_, err = parseWavString(t, "RIFF\x29\x00\x00\x00WAVEfmt \x10\x00\x00\x00\x01\x00\x05\x00\x00\x00\x01\x00\x00\x00\x05\x00\x05\x00\x08\x00dota\x05\x00\x00\x00hello")
	if err == nil {
		t.Errorf("bogus 'dota' block should not have parsed")
	}

	_, err = parseWavString(t, "RIFF\x29\x00\x00\x00WAVEfmt \x10\x00\x00\x00\x01\x00\x05\x00\x00\x00\x01\x00\x00\x00\x05\x00\x05\x00\x08\x00data\x0c\x00\x00\x00hello, world")
	if err == nil {
		t.Errorf("incorrect global/data lengths should not have parsed")
	}
}

type Buf struct {
	B *bytes.Buffer
}

func (buf Buf) Read(b []byte) (int, error) {
	return buf.B.Read(b)
}
func (buf Buf) Close() error {
	if buf.B == nil {
		return fmt.Errorf("already closed")
	}
	buf.B = nil
	return nil
}

func parseWavString(t *testing.T, wav string) (*Reader, error) {
	reader := Buf{bytes.NewBufferString(wav)}
	wc := New(reader)
	wc.Init()
	return wc, wc.errorState
}
