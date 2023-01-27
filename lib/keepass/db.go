package keepass

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

const (
	VERSION_NUMBER_LEN = 2
	TLV_TYPE_LEN       = 1
	TLV_LENGTH_LEN     = 2
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
	masterSeed         []byte
	transformSeed      []byte
	transformRounds    uint64
	encryptionIV       []byte
	protectedStreamKey []byte
	streamStartBytes   []byte
	irs                IRSID
}

type database struct {
	path              string
	content           string
	content_encrypted []byte
	verMajor          uint16
	verMinor          uint16
	headers           databaseHeaders
}

func NewDatabase(path string) database {
	return database{
		path:              path,
		content:           "",
		content_encrypted: make([]byte, 0),
		verMajor:          0,
		verMinor:          0,
		headers:           databaseHeaders{},
	}
}

func (d database) Content() string {
	return d.content
}

// Return kdbx version as tuple (major, minor)
func (d database) Version() (uint16, uint16) {
	return d.verMajor, d.verMinor
}

func (d *database) Load(password string) error {
	// Check if file exists
	_, err := os.Stat(d.path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("File '%s' does not exist", d.path)
		}
		return err
	}
	err = d.parse()
	if err != nil {
		return err
	}
	return nil
}

func (d *database) parse() error {
	f, err := os.Open(d.path)
	defer f.Close()
	if err != nil {
		return err
	}

	// Check filetype signature
	eq, err := readCompare(f, FILE_SIGNATURE[:])
	if err != nil {
		return err
	}
	if !eq {
		return errors.New("Invalid file signature")
	}

	// Check KeePass version signature
	eq, err = readCompare(f, VERSION_SIGNATURE[:])
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
			log.Printf("Skipping unknown header code: %d", htype)
		}
		headerMap[htype] = value
	}

	// Parse headers
	for _, h := range obligatoryHeaders {
		if _, present := headerMap[h]; !present {
			return fmt.Errorf("Missing header with code %d", h)
		}
	}

	if !bytes.Equal(headerMap[CipherID], AES_CIPHER_ID[:]) {
		return errors.New("Invalid or unsupported cipher")
	}

	if binary.LittleEndian.Uint32(headerMap[CompressionFlag]) != COMPRESSION_None {
		return errors.New("Gzip-compressed databases are not supported yet, sorry :(")
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
	d.content_encrypted, err = ioutil.ReadAll(f)
	if err != nil {
		return fmt.Errorf("Error while reading database content: %s", err)
	}

	return nil
}

// Read len(b) bytes from f and compare with b
func readCompare(f *os.File, b []byte) (bool, error) {
	buf := make([]byte, len(b))
	_, err := f.Read(buf)
	if err != nil {
		return false, err
	}
	return bytes.Equal(buf, b), nil
}
