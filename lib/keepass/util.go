package keepass

import (
	"bytes"
	"os"
)

// Read len(b) bytes from f and compare with b
func readCompare(f *os.File, b []byte) (bool, error) {
	buf := make([]byte, len(b))
	_, err := f.Read(buf)
	if err != nil {
		return false, err
	}
	return bytes.Equal(buf, b), nil
}
