package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
)

func AESRounds(in, seed []byte, rounds uint64) ([]byte, error) {
	if len(in)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("Input length must be multiple of block size %d", aes.BlockSize)
	}
	cfr, err := aes.NewCipher(seed)
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(in))
	copy(out, in[:])
	for i := uint64(0); i < rounds; i++ {
		for j := 0; j < len(out); j += aes.BlockSize {
			cfr.Encrypt(out[j:j+aes.BlockSize], out[j:j+aes.BlockSize])
		}
	}
	return out, nil
}

func DecryptAES(ciphertext, key, iv []byte) ([]byte, error) {
	cfr, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	plaintext := make([]byte, len(ciphertext))
	mode := cipher.NewCBCDecrypter(cfr, iv)
	mode.CryptBlocks(plaintext, ciphertext)
	return plaintext, nil
}
