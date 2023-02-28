package crypto

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCrypt(t *testing.T) {
	assert := assert.New(t)

	key := make([]byte, 16)
	iv := make([]byte, 16)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatal(err)
	}
	_, err = rand.Read(iv)
	if err != nil {
		t.Fatal(err)
	}

	plaintexts := [][]byte{
		[]byte("foo bar"),
		[]byte("Lorem ipsum dolor sit amet consectetur"),
	}
	for _, plaintext := range plaintexts {
		encrypted, err := EncryptAES(plaintext, key, iv)
		if !assert.Nil(err) {
			return
		}
		decrypted, err := DecryptAES(encrypted, key, iv)
		if !assert.Nil(err) {
			return
		}

		assert.Equal(plaintext, decrypted)
	}
}

func TestGenerateMasterKey(t *testing.T) {
	assert := assert.New(t)

	password := "foo"
	masterSeed := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	transformSeed := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	expectedMasterKey := []byte{0x45, 0x61, 0x38, 0x62, 0xc8, 0x59, 0x5f, 0x4e, 0x9b, 0x85, 0x3d, 0x10, 0xdc, 0xad, 0x69, 0x31, 0x3a, 0x9e, 0x69, 0x8e, 0x9d, 0x5a, 0x29, 0x1d, 0xda, 0x5d, 0x82, 0x84, 0xe0, 0xc7, 0x8f, 0x6c}

	masterKey, err := GenerateMasterKey(password, masterSeed, transformSeed, 10)
	if !assert.Nil(err) {
		return
	}

	assert.Equal(masterKey, expectedMasterKey)
}
