package util

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

// Read len(b) bytes from f and compare with b
func ReadCompare(f io.Reader, b []byte) (bool, error) {
	buf := make([]byte, len(b))
	_, err := f.Read(buf)
	if err != nil {
		return false, err
	}
	return bytes.Equal(buf, b), nil
}

// WriteAssert writes to f and returns an error if writing failed
func WriteAssert(f io.Writer, b []byte) error {
	n, err := f.Write(b)
	if err != nil {
		return fmt.Errorf("Error while writing to file: %s", err)
	}
	if n != len(b) {
		return fmt.Errorf("Tried to write %d bytes but wrote %d", len(b), n)
	}
	return nil
}

func Unzip(in *[]byte) (*[]byte, error) {
	out := make([]byte, 1024)
	var outBuf bytes.Buffer
	inBuf := bytes.NewBuffer(*in)
	r, err := gzip.NewReader(inBuf)
	if err != nil {
		return nil, err
	}
	for {
		n, err := r.Read(out)
		outBuf.Write(out[:n])
		if err == io.EOF || n == 0 {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	outBytes := outBuf.Bytes()
	return &outBytes, nil
}
