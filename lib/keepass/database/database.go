package database

import (
	"bytes"
	"crypto/aes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"

	"github.com/Zaphoood/tresor/lib/keepass/crypto"
	"github.com/Zaphoood/tresor/lib/keepass/parser"
	"github.com/Zaphoood/tresor/lib/keepass/util"
)

const (
	SHA256_DIGEST_LEN = 32
	WORD              = 2
	DWORD             = 4
)

type block struct {
	start  int
	length int
}

type version struct {
	major uint16
	minor uint16
}

func (v *version) read(r io.Reader) error {
	buf := make([]byte, WORD)

	read, err := r.Read(buf)
	if err != nil {
		return err
	}
	if read != len(buf) {
		return errors.New("File truncated")
	}
	v.minor = binary.LittleEndian.Uint16(buf)

	read, err = r.Read(buf)
	if err != nil {
		return err
	}
	if read != len(buf) {
		return errors.New("File truncated")
	}
	v.major = binary.LittleEndian.Uint16(buf)

	return nil
}

type Database struct {
	path       string
	version    version
	header     header
	headerHash [SHA256_DIGEST_LEN]byte

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

func (d Database) Version() version {
	return d.version
}

func (d *Database) Load() error {
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

	err = d.version.read(f)
	if err != nil {
		return err
	}

	err = d.header.read(f)
	if err != nil {
		return err
	}

	headerLength, err := f.Seek(0, 1)
	if err != nil {
		return err
	}
	headerRaw := make([]byte, headerLength)
	_, err = f.Seek(0, 0)
	if err != nil {
		return err
	}
	f.Read(headerRaw)
	d.headerHash = sha256.Sum256(headerRaw)

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
	masterKey, err := d.generateMasterKey(password)
	if err != nil {
		return err
	}

	plaintext, err := crypto.DecryptAES(d.ciphertext, masterKey, d.header.encryptionIV)
	if err != nil {
		return err
	}

	if !d.checkStreamStartBytes(&plaintext) {
		// TODO: Create custom error for this
		return errors.New("Wrong password")
	}

	err = d.parseBlocks(&plaintext)
	if err != nil {
		return err
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

func (d *Database) generateMasterKey(password string) ([]byte, error) {
	// Generate composite key
	compositeKey := sha256.Sum256([]byte(password))
	compositeKey = sha256.Sum256(compositeKey[:])

	// Generate master key
	transformOut, err := crypto.AESRounds(compositeKey[:], d.header.transformSeed, d.header.transformRounds)
	if err != nil {
		return nil, err
	}
	transformKey := sha256.Sum256(transformOut)

	h := sha256.New()
	h.Write(d.header.masterSeed)
	h.Write(transformKey[:])
	return h.Sum(nil), nil
}

func (d *Database) checkStreamStartBytes(plaintext *[]byte) bool {
	ok := bytes.Equal(d.header.streamStartBytes, (*plaintext)[:len(d.header.streamStartBytes)])
	*plaintext = (*plaintext)[len(d.header.streamStartBytes):]
	return ok
}

func (d *Database) parseBlocks(plaintextBlocks *[]byte) error {
	blocks := make(map[uint32]block)
	hashIndex := 0
	totalSize := 0
	i := 0
	for i < len((*plaintextBlocks)) {
		// Read block id
		blockID := binary.LittleEndian.Uint32((*plaintextBlocks)[i : i+DWORD])
		i += DWORD
		if _, exists := blocks[blockID]; exists {
			return fmt.Errorf("Duplicate block ID: %d", blockID)
		}
		// Store index of hash for later comparison
		hashIndex = i
		i += SHA256_DIGEST_LEN

		// Read block size
		blockSize := int(binary.LittleEndian.Uint32((*plaintextBlocks)[i : i+DWORD]))
		// Final block has block size 0
		if blockSize == 0 {
			break
		}
		i += DWORD

		// Hash and compare
		hash := sha256.Sum256((*plaintextBlocks)[i : i+blockSize])
		if !bytes.Equal(hash[:], (*plaintextBlocks)[hashIndex:hashIndex+SHA256_DIGEST_LEN]) {
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
		copy(d.plaintext[pos:pos+block.length], (*plaintextBlocks)[block.start:block.start+block.length])
		pos += block.length
	}

	return nil
}

func (d *Database) Parse() error {
	var err error
	d.parsed, err = parser.Parse(d.plaintext, d.header.protectedStreamKey)
	if err != nil {
		return err
	}

	return nil
}

func (d *Database) VerifyHeaderHash() (bool, error) {
	if len(d.parsed.Meta.HeaderHash) == 0 {
		return false, errors.New("No header hash found in XML")
	}
	storedHashEnc := []byte(d.parsed.Meta.HeaderHash)
	storedHash := make([]byte, base64.StdEncoding.DecodedLen(len(storedHashEnc)))
	length, err := base64.StdEncoding.Decode(storedHash, storedHashEnc)
	if err != nil {
		return false, err
	}
	return bytes.Equal(d.headerHash[:], storedHash[:length]), nil
}
