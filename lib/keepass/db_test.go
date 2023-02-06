package keepass

import (
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	XML_HEADER      = []byte("<?xml version=\"1.0\" encoding=\"utf-8\" standalone=\"yes\"?>")
	KEEPASS_END_TAG = []byte("</KeePassFile>")
)

func TestFileNotExist(t *testing.T) {
	d := NewDatabase("/this/path/does/not/exist.kdbx")
	err := d.Load()
	assert.NotNil(t, err, "Want error for non-existent path, got nil")
}

func TestLoadDb(t *testing.T) {
	assert := assert.New(t)

	d := NewDatabase("./test/example.kdbx")
	err := d.Load()
	assert.Nil(err)

	majorExpected, minorExpected := 3, 1
	major, minor := d.Version()
	assert.Equal(major, uint16(majorExpected), fmt.Sprintf("Expected major version: %d, actual: %d", major, majorExpected))
	assert.Equal(minor, uint16(minorExpected), fmt.Sprintf("Expected minor version: %d, actual: %d", minor, minorExpected))

	err = d.Decrypt("foo")
	assert.Nil(err)

	plaintext := d.Plaintext()
	assert.Equal(string(plaintext[:len(XML_HEADER)]), string(XML_HEADER))
	assert.Equal(string(plaintext[len(plaintext)-len(KEEPASS_END_TAG):]), string(KEEPASS_END_TAG))

	err = d.Parse()
	assert.Nil(err)

	valid, err := d.VerifyHeaderHash()
	assert.Nil(err)
	assert.True(valid, "Invalid header hash")
}

func TestInvalidFileSignature(t *testing.T) {
	d := NewDatabase("./test/invalid_file_signature.kdbx")
	err := d.Load()
	assert.NotNil(t, err, "Want error for file with invalid file signature, got nil")
}

func TestInvalidVersionSignature(t *testing.T) {
	d := NewDatabase("./test/invalid_version_signature.kdbx")
	err := d.Load()
	assert.NotNil(t, err, "Want error for file with invalid version signature, got nil")
}

func TestInvalidCipherID(t *testing.T) {
	d := NewDatabase("./test/invalid_cipher_id.kdbx")
	err := d.Load()
	assert.NotNil(t, err, "Want error for file with invalid cipher id, got nil")
}

func TestCompressed(t *testing.T) {
	// Compression is not implemented yet, so we want to return an error for compressed databases
	d := NewDatabase("./test/example_compressed.kdbx")
	err := d.Load()
	assert.Nil(t, err)

	err = d.Decrypt("foo")
	assert.Nil(t, err)
}

func TestInvalidCiphertextLength(t *testing.T) {
	d := NewDatabase("./test/invalid_length.kdbx")
	err := d.Load()
	assert.NotNil(t, err, "Want error for invalid cipher text length, got nil")
}

func TestInvalidStreamStartBytes(t *testing.T) {
	d := NewDatabase("./test/invalid_ssb.kdbx")
	err := d.Load()
	assert.Nil(t, err)

	err = d.Decrypt("foo")
	assert.NotNil(t, err, "Want error for invalid stream start bytes, got nil")
}

func TestTruncated(t *testing.T) {
	d := NewDatabase("./test/truncated.kdbx")
	err := d.Load()
	assert.Equal(t, err, io.EOF, "Want EOF for truncated file, got nil")
}
