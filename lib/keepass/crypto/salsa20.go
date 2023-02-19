package crypto

import (
	"golang.org/x/crypto/salsa20"
)

var SALSA20_NONCE []byte = []byte{0xE8, 0x30, 0x09, 0x4B, 0x97, 0x20, 0x5D, 0x2A}

type Salsa20Stream struct {
	accum []byte
	nonce []byte
	key   [32]byte
}

func NewSalsa20Stream(key [32]byte) *Salsa20Stream {
	return &Salsa20Stream{nonce: SALSA20_NONCE, key: key}
}

func (s *Salsa20Stream) Decrypt(ciphertext []byte) ([]byte, error) {
	s.accum = append(s.accum, ciphertext...)
	out := make([]byte, len(s.accum))
	salsa20.XORKeyStream(out, s.accum, s.nonce, &s.key)

	return out[len(out)-len(ciphertext):], nil
}


// Encrypt encrypts a given bytearray. The operation is the same as decrypting
func (s *Salsa20Stream) Encrypt(plaintext []byte) ([]byte, error) {
	return s.Decrypt(plaintext)
}
