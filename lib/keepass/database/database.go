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

const (
	VERSION_NUMBER_LEN = 2
	BLOCK_HASH_LEN     = 32
	DWORD_LEN          = 4
)

type block struct {
	start  int
	length int
}

type Database struct {
	path       string
	verMajor   uint16
	verMinor   uint16
	header     header
	header_raw []byte // Store entire headers here; verify hash after decrypting

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

func (d Database) Plaintext() []byte {
	return d.plaintext
}

func (d Database) Parsed() *parser.Document {
	return d.parsed
}

// Return kdbx version as tuple (major, minor)
func (d Database) Version() (uint16, uint16) {
	return d.verMajor, d.verMinor
}

func (d *Database) Load() error {
	// Make sure file exists
	_, err := os.Stat(d.path)
	if err != nil {
		return err
	}

	f, err := os.Open(d.path)
	defer f.Close()
	if err != nil {
		return err
	}

	// Check filetype signature
	eq, err := util.ReadCompare(f, FILE_SIGNATURE[:])
	if err != nil {
		return err
	}
	if !eq {
		return errors.New("Invalid file signature")
	}

	// Check KeePass version signature
	eq, err = util.ReadCompare(f, VERSION_SIGNATURE[:])
	if err != nil {
		return err
	}
	if !eq {
		return errors.New("Invalid or unsupported version signature")
	}

	// Read kdbx version
	bufMinor := make([]byte, VERSION_NUMBER_LEN)
	bufMajor := make([]byte, VERSION_NUMBER_LEN)
	read, err := f.Read(bufMinor)
	if err != nil {
		return err
	}
	if read != len(bufMinor) {
		return errors.New("File truncated")
	}
	read, err = f.Read(bufMajor)
	if err != nil {
		return err
	}
	if read != len(bufMajor) {
		return errors.New("File truncated")
	}
	d.verMinor = binary.LittleEndian.Uint16(bufMinor)
	d.verMajor = binary.LittleEndian.Uint16(bufMajor)

	// Read header
	err = d.header.read(f)
	if err != nil {
		return err
	}

	// Store raw header content for hashing later
	headersLength, err := f.Seek(0, 1)
	if err != nil {
		return err
	}
	d.header_raw = make([]byte, headersLength)
	_, err = f.Seek(0, 0)
	if err != nil {
		return err
	}
	f.Read(d.header_raw)

	// Read remaining file content
	d.ciphertext, err = ioutil.ReadAll(f)
	if err != nil {
		return fmt.Errorf("Error while reading database content: %s", err)
	}

	if len(d.ciphertext)%aes.BlockSize != 0 {
		return fmt.Errorf("Invalid cipher text: length must be multiple of block size %d", aes.BlockSize)
	}

	return nil
}

func (d *Database) Decrypt(password string) error {
	// Generate composite key
	compositeKey := sha256.Sum256([]byte(password))
	compositeKey = sha256.Sum256(compositeKey[:])

	// Generate master key
	transformOut, err := crypto.AESRounds(compositeKey[:], d.header.transformSeed, d.header.transformRounds)
	if err != nil {
		return err
	}
	transformKey := sha256.Sum256(transformOut)

	h := sha256.New()
	h.Write(d.header.masterSeed)
	h.Write(transformKey[:])
	masterKey := h.Sum(nil)

	// Decrypt content
	plaintext, err := crypto.DecryptAES(d.ciphertext, masterKey, d.header.encryptionIV)
	if err != nil {
		return err
	}

	// Verify that decrypting was successful
	if !d.checkStreamStartBytes(&plaintext) {
		return errors.New("Wrong password")
	}

	blocks := make(map[uint32]block)
	hashIndex := 0
	totalSize := 0
	i := 0
	for i < len(plaintext) {
		// Read block id
		blockID := binary.LittleEndian.Uint32(plaintext[i : i+DWORD_LEN])
		i += DWORD_LEN
		if _, exists := blocks[blockID]; exists {
			return fmt.Errorf("Duplicate block ID: %d", blockID)
		}
		// Store index of hash for later comparison
		hashIndex = i
		i += BLOCK_HASH_LEN

		// Read block size
		blockSize := int(binary.LittleEndian.Uint32(plaintext[i : i+DWORD_LEN]))
		// Final block has block size 0
		if blockSize == 0 {
			break
		}
		i += DWORD_LEN

		// Hash and compare
		hash := sha256.Sum256(plaintext[i : i+blockSize])
		if !bytes.Equal(hash[:], plaintext[hashIndex:hashIndex+BLOCK_HASH_LEN]) {
			return errors.New("Block hash does not match. File may be corrupted")
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

	d.plaintext = make([]byte, totalSize)
	pos := 0
	for _, id := range blockIDs {
		block := blocks[uint32(id)]
		copy(d.plaintext[pos:pos+block.length], plaintext[block.start:block.start+block.length])
		pos += block.length
	}

	if d.header.gzipCompression {
		out, err := util.Unzip(&d.plaintext)
		d.plaintext = *out
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Database) checkStreamStartBytes(plaintext *[]byte) bool {
	ok := bytes.Equal(d.header.streamStartBytes, (*plaintext)[:len(d.header.streamStartBytes)])
	*plaintext = (*plaintext)[len(d.header.streamStartBytes):]
	return ok
}

func (d *Database) Parse() error {
	var err error
	d.parsed, err = parser.Parse(d.plaintext)
	if err != nil {
		return err
	}

	return nil
}

func (d *Database) VerifyHeaderHash() (bool, error) {
	if len(d.parsed.Meta.HeaderHash) == 0 {
		return false, errors.New("No header hash found")
	}
	storedHashEnc := []byte(d.parsed.Meta.HeaderHash)
	storedHash := make([]byte, base64.StdEncoding.DecodedLen(len(storedHashEnc)))
	_, err := base64.StdEncoding.Decode(storedHash, storedHashEnc)
	if err != nil {
		return false, err
	}
	actualHash := sha256.Sum256(d.header_raw)
	// storedHash may be too long, since its length is taken from base64.StdEncoding.DecodedLen
	// Therefore we only compare the first len(actualHash) bytes
	return bytes.Equal(actualHash[:], storedHash[:len(actualHash)]), nil
}
