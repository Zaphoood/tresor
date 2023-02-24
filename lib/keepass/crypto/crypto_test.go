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
