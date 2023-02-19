package crypto

type Stream interface {
	Decrypt(in []byte) (out []byte, err error)
	Encrypt(in []byte) (out []byte, err error)
}
