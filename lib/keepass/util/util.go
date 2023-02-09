package util

import (
	"bytes"
	"compress/gzip"
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
