package database

import (
	"fmt"
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
		{"../test/example_compressed.kdbx", "foo"},
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

func TestErrors(t *testing.T) {
	cases := []struct {
		path       string
		loadErr    error
		decryptErr error
	}{
		{"../test/invalid_file_signature.kdbx", FileError{}, nil},
		{"../test/invalid_version_signature.kdbx", FileError{}, nil},
		{"../test/invalid_cipher_id.kdbx", FileError{}, nil},
		{"../test/invalid_length.kdbx", FileError{}, nil},
		{"../test/invalid_ssb.kdbx", nil, DecryptError{}},
	}
	for _, c := range cases {
		d := New(c.path)
		err := d.Load()
		assert.IsType(t, c.loadErr, err, fmt.Sprintf("Expected '%T' when loading '%s'", c.loadErr, c.path))
		if err == nil {
			err = d.Decrypt("foo")
			assert.IsType(t, c.decryptErr, err, fmt.Sprintf("Expected '%T' when decrypting '%s'", c.decryptErr, c.path))
		}
	}
}
