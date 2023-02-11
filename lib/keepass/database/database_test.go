package database

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
	d := New("/this/path/does/not/exist.kdbx")
	err := d.Load()
	assert.NotNil(t, err, "Want error for non-existent path, got nil")
}

func TestLoadDb(t *testing.T) {
	assert := assert.New(t)

	files := []struct {
		path     string
		password string
	}{
		//{"../test/example_compressed.kdbx", "foo"},
		{"../test/example.kdbx", "foo"},
	}
	for _, file := range files {
		d := New(file.path)
		err := d.Load()
		assert.Nil(err)

		expectedVersion := version{3, 1}
		version := d.Version()
		assert.Equal(expectedVersion, version, fmt.Sprintf("Expected version: %d, got: %d", expectedVersion, version))

		err = d.Decrypt(file.password)
		assert.Nil(err)

		plaintext := d.Plaintext()
		assert.Equal(string(XML_HEADER), string(plaintext[:len(XML_HEADER)]))
		assert.Equal(string(KEEPASS_END_TAG), string(plaintext[len(plaintext)-len(KEEPASS_END_TAG):]))

		err = d.Parse()
		assert.Nil(err)

		valid, err := d.VerifyHeaderHash()
		assert.Nil(err)
		assert.True(valid, "Invalid header hash")
	}
}

func TestInvalidFileSignature(t *testing.T) {
	d := New("../test/invalid_file_signature.kdbx")
	err := d.Load()
	if assert.NotNil(t, err) {
		assert.IsType(t, FileError{}, err)
	}
}

func TestInvalidVersionSignature(t *testing.T) {
	d := New("../test/invalid_version_signature.kdbx")
	err := d.Load()
	if assert.NotNil(t, err) {
		assert.IsType(t, FileError{}, err)
	}

}

func TestInvalidCipherID(t *testing.T) {
	d := New("../test/invalid_cipher_id.kdbx")
	err := d.Load()
	if assert.NotNil(t, err) {
		assert.IsType(t, FileError{}, err)
	}
}

func TestInvalidCiphertextLength(t *testing.T) {
	d := New("../test/invalid_length.kdbx")
	err := d.Load()
	if assert.NotNil(t, err) {
		assert.IsType(t, DecryptError{}, err)
	}
}

func TestInvalidStreamStartBytes(t *testing.T) {
	d := New("../test/invalid_ssb.kdbx")
	err := d.Load()
	assert.Nil(t, err)

	err = d.Decrypt("foo")
	if assert.NotNil(t, err) {
		assert.IsType(t, DecryptError{}, err)
	}
}

func TestTruncated(t *testing.T) {
	d := New("../test/truncated.kdbx")
	err := d.Load()
	assert.Equal(t, io.EOF, err)
}
