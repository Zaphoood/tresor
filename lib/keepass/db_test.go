package keepass

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

var (
	XML_HEADER      = []byte("<?xml version=\"1.0\" encoding=\"utf-8\" standalone=\"yes\"?>")
	KEEPASS_END_TAG = []byte("</KeePassFile>")
)

func TestFileNotExist(t *testing.T) {
	d := NewDatabase("/this/path/does/not/exist.kdbx")
	err := d.Load()
	if err == nil {
		t.Error("Want error for non-existent path, got nil")
	}
}

func TestLoadDb(t *testing.T) {
	d := NewDatabase("./test/example.kdbx")
	err := d.Load()
	if err != nil {
		t.Fatal(err)
	}

	expectedMajor := uint16(3)
	expectedMinor := uint16(1)
	if major, minor := d.Version(); major != expectedMajor || minor != expectedMinor {
		t.Errorf("Want version to be %d.%d but got %d.%d", expectedMajor, expectedMinor, major, minor)
	}

	err = d.Decrypt("foo")
	if err != nil {
		t.Error(err)
	}

	content := d.Content()
	if !bytes.Equal(content[:len(XML_HEADER)], XML_HEADER) {
		t.Error(fmt.Sprintf("Expected XML header:\n%s\ngot:\n%s", XML_HEADER, content[:len(XML_HEADER)]))
	}
	if !bytes.Equal(content[len(content)-len(KEEPASS_END_TAG):], KEEPASS_END_TAG) {
		t.Error(fmt.Sprintf("Expected end tag to be:\n%s\ngot:\n%s", KEEPASS_END_TAG, content[len(content)-len(KEEPASS_END_TAG):]))
	}
}

func TestInvalidFileSignature(t *testing.T) {
	d := NewDatabase("./test/invalid_file_signature.kdbx")
	err := d.Load()
	if err == nil {
		t.Fatal("Want error for file with invalid file signature, got nil")
	}
}

func TestInvalidVersionSignature(t *testing.T) {
	d := NewDatabase("./test/invalid_version_signature.kdbx")
	err := d.Load()
	if err == nil {
		t.Fatal("Want error for file with invalid version signature, got nil")
	}
}

func TestInvalidCipherID(t *testing.T) {
	d := NewDatabase("./test/invalid_cipher_id.kdbx")
	err := d.Load()
	if err == nil {
		t.Fatal("Want error for file with invalid cipher id, got nil")
	}
}

func TestCompressed(t *testing.T) {
	// Compression is not implemented yet, so we want to return an error for compressed databases
	d := NewDatabase("./test/compressed.kdbx")
	err := d.Load()
	if err == nil {
		t.Fatal("Want error for compressed database, got nil")
	}
}

func TestInvalidCiphertextLength(t *testing.T) {
	d := NewDatabase("./test/invalid_length.kdbx")

	err := d.Load()
	if err == nil {
		t.Fatal("Want error for invalid cipher text length, got nil")
	}
}

func TestInvalidStreamStartBytes(t *testing.T) {
	d := NewDatabase("./test/invalid_ssb.kdbx")
	err := d.Load()
	if err != nil {
		t.Fatal(err)
	}

	err = d.Decrypt("foo")
	if err == nil {
		t.Fatal("Want error for invalid stream start bytes, got nil")
	}
}

func TestTruncated(t *testing.T) {
	d := NewDatabase("./test/truncated.kdbx")
	err := d.Load()
	if err != io.EOF {
		t.Fatal("Want EOF for truncated file, got nil")
	}
}
