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
	"log"
	"os"
	"sort"

	"github.com/Zaphoood/tresor/lib/keepass/crypto"
	"github.com/Zaphoood/tresor/lib/keepass/parser"
	"github.com/Zaphoood/tresor/lib/keepass/util"
)

const (
	VERSION_NUMBER_LEN = 2
	TLV_TYPE_LEN       = 1
	TLV_LENGTH_LEN     = 2
	BLOCK_HASH_LEN     = 32
	DWORD_LEN          = 4
)

type headerCode uint8

const (
	// End of headers
	EOH headerCode = iota
	Comment
	CipherID
	CompressionFlag
	MasterSeed
	TransformSeed
	TransformRounds
	EncryptionIV
	ProtectedStreamKey
	StreamStartBytes
	InnerRandomStreamID
	// Store number of header codes so that we can iterate
	NUM_HEADER_CODES
)

// These headers need to be present in order for us to open the database
var obligatoryHeaders [9]headerCode = [9]headerCode{
	CipherID,
	CompressionFlag,
	MasterSeed,
	TransformSeed,
	TransformRounds,
	EncryptionIV,
	ProtectedStreamKey,
	StreamStartBytes,
	InnerRandomStreamID,
}

func validHeaderCode(c headerCode) bool {
	return EOH <= c && c < NUM_HEADER_CODES
}

var (
	FILE_SIGNATURE    [4]byte  = [4]byte{0x03, 0xD9, 0xA2, 0x9A}
	VERSION_SIGNATURE [4]byte  = [4]byte{0x67, 0xFB, 0x4B, 0xB5}
	AES_CIPHER_ID     [16]byte = [16]byte{0x31, 0xC1, 0xF2, 0xE6, 0xBF, 0x71, 0x43, 0x50, 0xBE, 0x58, 0x05, 0x21, 0x6A, 0xFC, 0x5A, 0xFF}
)

const (
	COMPRESSION_None = 0
	COMPRESSION_GZip = 1
)

type IRSID int

const (
	IRS_None = iota
	IRS_ARC4
	IRS_Salsa20
)

func validIRSID(id uint32) bool {
	return IRS_None <= id && id <= IRS_Salsa20
}

type databaseHeaders struct {
	gzipCompression    bool
	masterSeed         []byte
	transformSeed      []byte
	transformRounds    uint64
	encryptionIV       []byte
	protectedStreamKey []byte
	streamStartBytes   []byte
	irs                IRSID
}

type block struct {
	start  int
	length int
}

type Database struct {
	path        string
	verMajor    uint16
	verMinor    uint16
	headers     databaseHeaders
	headers_raw []byte // Store entire headers here; verify hash after decrypting

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

	// Read headers
	headerMap := make(map[headerCode][]byte)
	bufType := make([]byte, TLV_TYPE_LEN)
	bufLength := make([]byte, TLV_LENGTH_LEN)
	var (
		htype  headerCode
		length uint16
		value  []byte
	)
	for {
		read, err = f.Read(bufType)
		if err != nil {
			return err
		}
		if read != len(bufType) {
			return errors.New("File truncated")
		}
		read, err = f.Read(bufLength)
		if err != nil {
			return err
		}
		if read != len(bufLength) {
			return errors.New("File truncated")
		}
		htype = headerCode(bufType[0])
		length = binary.LittleEndian.Uint16(bufLength)
		value = make([]byte, length)
		read, err = f.Read(value)
		if htype == EOH {
			break
		}
		if !validHeaderCode(htype) {
			log.Printf("WARNING: Skipping invalid header code: %d", htype)
		}
		headerMap[htype] = value
	}

	// Store headers for later hashing
	headersLength, err := f.Seek(0, 1)
	if err != nil {
		return err
	}
	d.headers_raw = make([]byte, headersLength)
	_, err = f.Seek(0, 0)
	if err != nil {
		return err
	}
	f.Read(d.headers_raw)

	// Parse headers
	for _, h := range obligatoryHeaders {
		if _, present := headerMap[h]; !present {
			return fmt.Errorf("Missing header with code %d", h)
		}
	}

	if !bytes.Equal(headerMap[CipherID], AES_CIPHER_ID[:]) {
		return errors.New("Invalid or unsupported cipher")
	}

	switch flag := binary.LittleEndian.Uint32(headerMap[CompressionFlag]); flag {
	case COMPRESSION_None:
		d.headers.gzipCompression = false
	case COMPRESSION_GZip:
		d.headers.gzipCompression = true
	default:
		return fmt.Errorf("Unknown compression flag: %d", flag)
	}

	d.headers.masterSeed = headerMap[MasterSeed]
	d.headers.transformSeed = headerMap[TransformSeed]
	d.headers.transformRounds = binary.LittleEndian.Uint64(headerMap[TransformRounds])
	d.headers.encryptionIV = headerMap[EncryptionIV]
	d.headers.protectedStreamKey = headerMap[ProtectedStreamKey]
	d.headers.streamStartBytes = headerMap[StreamStartBytes]

	irsid := binary.LittleEndian.Uint32(headerMap[InnerRandomStreamID])
	if !validIRSID(irsid) {
		return fmt.Errorf("Invalid Inner Random Stream ID: %d", irsid)
	}
	d.headers.irs = IRSID(irsid)

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
	transformOut, err := crypto.AESRounds(compositeKey[:], d.headers.transformSeed, d.headers.transformRounds)
	if err != nil {
		return err
	}
	transformKey := sha256.Sum256(transformOut)

	h := sha256.New()
	h.Write(d.headers.masterSeed)
	h.Write(transformKey[:])
	masterKey := h.Sum(nil)

	// Decrypt content
	plaintext, err := crypto.DecryptAES(d.ciphertext, masterKey, d.headers.encryptionIV)
	if err != nil {
		return err
	}

	// Verify that decrypting was successful
	if !bytes.Equal(d.headers.streamStartBytes, plaintext[:len(d.headers.streamStartBytes)]) {
		return errors.New("Wrong password")
	}
	plaintext = plaintext[len(d.headers.streamStartBytes):]

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

	if d.headers.gzipCompression {
		out, err := util.Unzip(&d.plaintext)
		d.plaintext = *out
		if err != nil {
			return err
		}
	}

	return nil
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
	actualHash := sha256.Sum256(d.headers_raw)
	// storedHash may be too long, since its length is taken from base64.StdEncoding.DecodedLen
	// Therefore we only compare the first len(actualHash) bytes
	return bytes.Equal(actualHash[:], storedHash[:len(actualHash)]), nil
}
