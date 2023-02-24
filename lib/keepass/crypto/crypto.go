package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"fmt"

	"github.com/andreburgaud/crypt2go/padding"
)

type DecryptError error

func GenerateMasterKey(password string, masterSeed, transformSeed []byte, transformRounds uint64) ([]byte, error) {
	// Generate composite key
	compositeKey := sha256.Sum256([]byte(password))
	compositeKey = sha256.Sum256(compositeKey[:])

	// Generate master key
	transformOut, err := AESRounds(compositeKey[:], transformSeed, transformRounds)
	if err != nil {
		return nil, err
	}
	transformKey := sha256.Sum256(transformOut)

	h := sha256.New()
	h.Write(masterSeed)
	h.Write(transformKey[:])
	return h.Sum(nil), nil
}

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

	padder := padding.NewPkcs7Padding(aes.BlockSize)
	unpadded, err := padder.Unpad(plaintext)
	if err != nil {
		return nil, err
	} else {
		return unpadded, nil
	}
}

func EncryptAES(plaintext, key, iv []byte) ([]byte, error) {
	cfr, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	padder := padding.NewPkcs7Padding(aes.BlockSize)
	padded, err := padder.Pad(plaintext)
	if err != nil {
		return nil, err
	}
	ciphertext := make([]byte, len(padded))
	mode := cipher.NewCBCEncrypter(cfr, iv)
	mode.CryptBlocks(ciphertext, padded)
	return ciphertext, nil
}
