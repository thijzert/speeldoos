package wavreader

import "testing"

func TestAtoI(t *testing.T) {
	b := []byte{1, 0, 0, 128}
	tatoi(t, b[2:4], 1<<15)
	tatoi(t, b[1:4], 1<<23)
	tatoi(t, b, 1<<31+1)
}

func tatoi(t *testing.T, buf []byte, exp int) {
	i := atoi(buf)
	if i != exp {
		t.Errorf("atoi failed for %02x: expected %d (%02x); got %d (%02x)", buf, exp, exp, i, i)
	}
}

func TestAtoSI(t *testing.T) {
	b := []byte{0, 0, 0, 128}
	tatosi(t, b[2:4], -1*1<<15)
	tatosi(t, b[1:4], -1*1<<23)
}

func tatosi(t *testing.T, buf []byte, exp int) {
	i := atosi(buf)
	if i != exp {
		t.Errorf("atosi failed for %02x: expected %d (%02x); got %d (%02x)", buf, exp, exp, i, i)
	}
}
