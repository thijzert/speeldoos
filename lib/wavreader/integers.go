package wavreader

// byte array to unsigned integer
func atoi(buf []byte) int {
	var rv int = 0
	for i, b := range buf {
		rv |= int(b) << uint(i*8)
	}
	return rv
}

// byte array to signed integer
func atosi(buf []byte) int {
	var rv int = 0
	for i, b := range buf {
		rv |= int(b) << uint(i*8)
	}

	if len(buf) < 8 {
		if buf[len(buf)-1]&0x80 != 0 {
			rv -= 1 << uint(8*len(buf))
		}
	}
	return rv
}

// unsigned integer to byte array
func itoa(buf []byte, v int) {
	// Truncate overflows
	if v < 0 {
		v = 0
	} else if v >= (1 << uint(len(buf)*8)) {
		v = (1 << uint(len(buf)*8)) - 1
	}

	for i := range buf {
		buf[i] = byte(v & 0xff)
		v >>= 8
	}
}

// signed integer to byte array
func sitoa(buf []byte, v int) {
	if v < 0 {
		if v < -1*(1<<uint(len(buf)*8-1)) {
			v = -1 * (1 << uint(len(buf)*8-1))
		}

		v += 1 << uint(len(buf)*8)
	} else {
		if v >= (1 << uint(len(buf)*8-1)) {
			v = (1 << uint(len(buf)*8-1)) - 1
		}
	}

	itoa(buf, v)
}
