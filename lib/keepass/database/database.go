package database

import (
	"bytes"
	"crypto/aes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"

	"github.com/Zaphoood/tresor/lib/keepass/crypto"
	"github.com/Zaphoood/tresor/lib/keepass/parser"
	"github.com/Zaphoood/tresor/lib/keepass/util"
)

const (
	WORD  = 2
	DWORD = 4
	QWORD = 8
)

type block struct {
	start  int
	length int
}

type FileError struct {
	err error
}

func (e FileError) Error() string {
	return e.err.Error()
}

type ParseError struct {
	err error
}

func (e ParseError) Error() string {
	return e.err.Error()
}

type DecryptError struct {
	err error
}

func (e DecryptError) Error() string {
	return e.err.Error()
}

type BlockSizeError struct {
	expectedBlockSize int
}

func (e BlockSizeError) Error() string {
	return fmt.Sprintf("Invalid cipher text: length must be multiple of block size %d", e.expectedBlockSize)
}

type Database struct {
	path     string
	password string
	header   header

	ciphertext []byte
	plaintext  []byte
	parsed     *parser.Document
}

func New(path string) Database {
	return Database{
		path: path,
	}
}

func (d Database) Path() string {
	return d.path
}

func (d *Database) SetPassword(password string) {
	d.password = password
}

func (d Database) Plaintext() []byte {
	return d.plaintext
}

func (d Database) Parsed() *parser.Document {
	return d.parsed
}

func (d Database) Version() version {
	return d.header.version
}

func (d *Database) Load() error {
	f, err := os.Open(d.path)
	defer f.Close()
	if err != nil {
		return err
	}

	err = d.header.read(f)
	if err != nil {
		return err
	}

	d.ciphertext, err = ioutil.ReadAll(f)
	if err != nil {
		return FileError{fmt.Errorf("Error while reading database content: %s", err)}
	}

	if len(d.ciphertext)%aes.BlockSize != 0 {
		return FileError{BlockSizeError{aes.BlockSize}}
	}

	return nil
}

func (d *Database) Decrypt() error {
	masterKey, err := crypto.GenerateMasterKey(d.password, d.header.masterSeed, d.header.transformSeed, d.header.transformRounds)
	if err != nil {
		return err
	}

	plaintext, err := crypto.DecryptAES(d.ciphertext, masterKey, d.header.encryptionIV)
	if err != nil {
		return err
	}

	if !d.checkStreamStartBytesAndTrim(&plaintext) {
		return DecryptError{errors.New("Wrong password")}
	}

	plaintext, err = parseBlocks(plaintext)
	if err != nil {
		return err
	}
	d.plaintext = plaintext

	if d.header.compression {
		d.plaintext, err = util.GUnzip(d.plaintext)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Database) checkStreamStartBytesAndTrim(plaintext *[]byte) bool {
	ok := bytes.Equal(d.header.streamStartBytes, (*plaintext)[:len(d.header.streamStartBytes)])
	*plaintext = (*plaintext)[len(d.header.streamStartBytes):]
	return ok
}

func parseBlocks(plainBlocks []byte) ([]byte, error) {
	in := bytes.NewReader(plainBlocks)
	var out bytes.Buffer
	blockCounter := uint32(0)
	for {
		buf := make([]byte, DWORD)
		err := util.ReadAssert(in, buf)
		if err != nil {
			return nil, err
		}
		blockID := binary.LittleEndian.Uint32(buf)
		if blockID != blockCounter {
			return nil, ParseError{fmt.Errorf("Invalid block ID: %d", blockID)}
		}
		blockCounter++

		storedHash := make([]byte, sha256.Size)
		util.ReadAssert(in, storedHash)

		buf = make([]byte, DWORD)
		err = util.ReadAssert(in, buf)
		if err != nil {
			return nil, err
		}
		blockSize := int(binary.LittleEndian.Uint32(buf))
		if blockSize == 0 {
			for _, b := range storedHash {
				if b != 0 {
					return nil, errors.New("Hash of final block must be zero")
				}
			}
			break
		}

		content := make([]byte, blockSize)
		util.ReadAssert(in, content)

		hash := sha256.Sum256(content)
		if !bytes.Equal(storedHash, hash[:]) {
			return nil, ParseError{errors.New("Block hash does not match. File may be corrupted")}
		}
		out.Write(content)
	}

	return out.Bytes(), nil
}

// formatBlocks formats the given byte array into blocks as per the kdbx file standard
func formatBlocks(in []byte) ([]byte, error) {
	var outBuf bytes.Buffer

	index := 0
	blockID := 0
	for {
		blockLength := len(in) - index
		if blockLength > math.MaxInt32 {
			blockLength = math.MaxInt32
		}
		block := in[index : index+blockLength]
		var hash [32]byte
		if index < len(in) {
			hash = sha256.Sum256(block)
		} else {
			// Last block's hash is all zeros, no need to do anything
		}
		blockIDBuf := make([]byte, DWORD)
		binary.LittleEndian.PutUint32(blockIDBuf, uint32(blockID))
		lengthBuf := make([]byte, DWORD)
		binary.LittleEndian.PutUint32(lengthBuf, uint32(blockLength))

		outBuf.Write(blockIDBuf)
		outBuf.Write(hash[:])
		outBuf.Write(lengthBuf[:])
		outBuf.Write(block)

		if index >= len(in) {
			break
		}

		index += blockLength
		blockID++
	}

	return outBuf.Bytes(), nil
}

func (d *Database) Parse() error {
	var err error
	d.parsed, err = parser.Parse(d.plaintext, sha256.Sum256(d.header.innerRandomStreamKey))
	if err != nil {
		return err
	}

	return nil
}

func (d *Database) VerifyHeaderHash() (bool, error) {
	if len(d.parsed.Meta.HeaderHash) == 0 {
		return false, errors.New("No header hash found in XML")
	}
	storedHash, err := base64.StdEncoding.DecodeString(d.parsed.Meta.HeaderHash)
	if err != nil {
		return false, err
	}
	return bytes.Equal(d.header.hashOfRead[:], storedHash[:]), nil
}

func (d *Database) Save() error {
	return d.SaveToPath(d.path)
}

func (d *Database) SaveToPath(path string) error {
	if d.parsed == nil {
		return errors.New("parsed must not be nil")
	}
	header := d.header.Copy()
	header.randomize()

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	hash, err := header.write(f)
	if err != nil {
		return err
	}

	d.parsed.Meta.HeaderHash = base64.StdEncoding.EncodeToString(hash[:])

	xml, err := parser.Unparse(d.parsed, sha256.Sum256(header.innerRandomStreamKey))
	if err != nil {
		return err
	}

	if header.compression {
		xml, err = util.GZip(xml)
		if err != nil {
			return err
		}
	}

	plainBlocks, err := formatBlocks(xml)
	if err != nil {
		return err
	}

	masterKey, err := crypto.GenerateMasterKey(d.password, header.masterSeed, header.transformSeed, d.header.transformRounds)
	if err != nil {
		return err
	}

	plaintext := make([]byte, 0, len(header.streamStartBytes)+len(plainBlocks))
	plaintext = append(plaintext, header.streamStartBytes...)
	plaintext = append(plaintext, plainBlocks...)
	ciphertext, err := crypto.EncryptAES(plaintext, masterKey, header.encryptionIV)
	if err != nil {
		return err
	}
	err = util.WriteAssert(f, ciphertext)

	return err
}
