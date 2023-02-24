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
	"os"
	"sort"

	"github.com/Zaphoood/tresor/lib/keepass/crypto"
	"github.com/Zaphoood/tresor/lib/keepass/parser"
	"github.com/Zaphoood/tresor/lib/keepass/util"
)

// TODO: Consider moving constants to a separate file
const (
	SHA256_DIGEST_LEN = 32
	WORD              = 2
	DWORD             = 4
	QWORD             = 8
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
	blocks := make(map[uint32]block)
	hashIndex := 0
	totalSize := 0
	i := 0
	for i < len(plainBlocks) {
		// Read block id
		blockID := binary.LittleEndian.Uint32(plainBlocks[i : i+DWORD])
		i += DWORD
		if _, exists := blocks[blockID]; exists {
			return nil, ParseError{fmt.Errorf("Duplicate block ID: %d", blockID)}
		}
		// Store index of hash for later comparison
		hashIndex = i
		i += SHA256_DIGEST_LEN

		// Read block size
		blockSize := int(binary.LittleEndian.Uint32(plainBlocks[i : i+DWORD]))
		// Final block has block size 0
		if blockSize == 0 {
			break
		}
		i += DWORD

		// Hash and compare
		hash := sha256.Sum256(plainBlocks[i : i+blockSize])
		if !bytes.Equal(hash[:], plainBlocks[hashIndex:hashIndex+SHA256_DIGEST_LEN]) {
			return nil, ParseError{errors.New("Block hash does not match. File may be corrupted")}
		}
		blocks[blockID] = block{start: i, length: blockSize}
		totalSize += blockSize
		i += blockSize
	}

	// Concatenate blocks by order of their IDs
	blockIDs := []int{}
	for id := range blocks {
		blockIDs = append(blockIDs, int(id))
	}
	sort.Ints(blockIDs)

	out := make([]byte, totalSize)
	pos := 0
	for _, id := range blockIDs {
		block := blocks[uint32(id)]
		copy(out[pos:pos+block.length], plainBlocks[block.start:block.start+block.length])
		pos += block.length
	}

	return out, nil
}

// formatBocks does the opposite of parseBlocks: It formats the given byte array as one block,
// followed by a zero-block which indicates the end of blocks
func formatBocks(in []byte) ([]byte, error) {
	zeroBuf := []byte{0x00, 0x00, 0x00, 0x00}
	oneBuf := []byte{0x01, 0x00, 0x00, 0x00}

	// TODO: Split input into multiple blocks if necessary
	inputLength := uint32(len(in))
	totalLength := 4*DWORD + 2*SHA256_DIGEST_LEN + inputLength
	hash := sha256.Sum256(in)
	lengthBuf := make([]byte, DWORD)
	binary.LittleEndian.PutUint32(lengthBuf, inputLength)

	out := make([]byte, 0, totalLength)
	// One block for all content
	out = append(out, zeroBuf...)
	out = append(out, hash[:]...)
	out = append(out, lengthBuf[:]...)
	out = append(out, in...)
	// Last block to signal end of file
	out = append(out, oneBuf...)
	out = append(out, make([]byte, SHA256_DIGEST_LEN)...)
	out = append(out, zeroBuf...)

	return out, nil
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

	plainBlocks, err := formatBocks(xml)
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
