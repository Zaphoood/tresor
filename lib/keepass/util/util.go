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

// WriteAssert writes to f and errors if writing failed or if it wasn't possible to write all bytes
func WriteAssert(w io.Writer, b []byte) error {
	n, err := w.Write(b)
	if err != nil {
		return fmt.Errorf("Error while writing to file: %s", err)
	}
	if n != len(b) {
		return fmt.Errorf("Writing failed: tried to write %d bytes but wrote only %d", len(b), n)
	}
	return nil
}

// ReadAssert tries to read len(b) bytes and errors if reading failed or if less bytes were read
func ReadAssert(r io.Reader, b []byte) error {
	n, err := r.Read(b)
	if err != nil {
		return err
	}
	if n != len(b) {
		return fmt.Errorf("File truncated: tried to read %d bytes but got only %d", len(b), n)
	}
	return nil
}

func GUnzip(in []byte) ([]byte, error) {
	out := make([]byte, 1024)
	var outBuf bytes.Buffer
	inBuf := bytes.NewBuffer(in)
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

	return outBuf.Bytes(), nil
}

func GZip(in []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	err := WriteAssert(writer, in)
	if err != nil {
		return nil, err
	}
	writer.Close()

	return buf.Bytes(), nil
}
