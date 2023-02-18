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
		if !assert.Nil(err) {
			continue
		}

		expectedVersion := version{3, 1}
		version := d.Version()
		assert.Equal(expectedVersion, version, fmt.Sprintf("Expected version: %d, got: %d", expectedVersion, version))

		d.SetPassword(file.password)
		err = d.Decrypt()
		if !assert.Nil(err) {
			continue
		}

		plaintext := d.Plaintext()
		assert.Equal(string(XML_HEADER), string(plaintext[:len(XML_HEADER)]))
		assert.Equal(string(KEEPASS_END_TAG), string(plaintext[len(plaintext)-len(KEEPASS_END_TAG):]))

		err = d.Parse()
		if !assert.Nil(err) {
			continue
		}

		valid, err := d.VerifyHeaderHash()
		if !assert.Nil(err) {
			continue
		}
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
	}
	for _, c := range cases {
		d := New(c.path)
		err := d.Load()
		assert.IsType(t, c.loadErr, err, fmt.Sprintf("Expected '%T' when loading '%s'", c.loadErr, c.path))
		if err == nil {
			d.SetPassword("foo")
			err = d.Decrypt()
			assert.IsType(t, c.decryptErr, err, fmt.Sprintf("Expected '%T' when decrypting '%s', got '%s'", c.decryptErr, c.path, err.Error()))
		}
	}
}
