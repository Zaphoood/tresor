package database

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/Zaphoood/tresor/src/keepass/util"
)

const (
	TLV_TYPE_LEN                = 1
	TLV_LENGTH_LEN              = 2
	MASTER_SEED_LEN             = 32
	TRANSFORM_SEED_LEN          = 32
	INNER_RANDOM_STREAM_KEY_LEN = 32
	STREAM_START_BYTES_LEN      = 32
	MAX_UINT16                  = ^uint16(0)
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

// These fields need to be present in order for us to open the database
var obligatoryFields [9]headerCode = [9]headerCode{
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
	// KeePass files contain this sequence as the data for the final header field, we just copy that behavior
	EOH_DATA [4]byte = [4]byte{0x0d, 0x0a, 0x0d, 0x0a}
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

type version struct {
	major uint16
	minor uint16
}

func (v *version) read(r io.Reader) error {
	buf := make([]byte, WORD)

	err := util.ReadAssert(r, buf)
	if err != nil {
		return err
	}
	v.minor = binary.LittleEndian.Uint16(buf)

	err = util.ReadAssert(r, buf)
	if err != nil {
		return err
	}
	v.major = binary.LittleEndian.Uint16(buf)

	return nil
}

func (v *version) write(w io.Writer) error {
	buf := make([]byte, WORD)
	binary.LittleEndian.PutUint16(buf, v.minor)
	err := util.WriteAssert(w, buf)
	if err != nil {
		return err
	}
	binary.LittleEndian.PutUint16(buf, v.major)
	err = util.WriteAssert(w, buf)
	if err != nil {
		return err
	}

	return nil
}

type header struct {
	version              version
	compression          bool
	masterSeed           []byte
	transformSeed        []byte
	transformRounds      uint64
	encryptionIV         []byte
	innerRandomStreamKey []byte
	streamStartBytes     []byte
	irsid                IRSID
	// Hash of the raw data that was read from disk during call to `header.read()`
	hashOfRead [sha256.Size]byte
}

func (h *header) Copy() *header {
	newHeader := header{
		version:              h.version,
		compression:          h.compression,
		masterSeed:           make([]byte, len(h.masterSeed)),
		transformSeed:        make([]byte, len(h.transformSeed)),
		transformRounds:      h.transformRounds,
		encryptionIV:         make([]byte, len(h.encryptionIV)),
		innerRandomStreamKey: make([]byte, len(h.innerRandomStreamKey)),
		streamStartBytes:     make([]byte, len(h.streamStartBytes)),
		irsid:                h.irsid,
	}
	copy(newHeader.masterSeed, h.masterSeed)
	copy(newHeader.transformSeed, h.transformSeed)
	copy(newHeader.encryptionIV, h.encryptionIV)
	copy(newHeader.innerRandomStreamKey, h.innerRandomStreamKey)
	copy(newHeader.streamStartBytes, h.streamStartBytes)

	return &newHeader
}

func newHeader(versionMajor int, versionMinor int, compression bool, transformRounds uint64, irsid IRSID, encryptionIVLength int) header {
	return header{
		compression:          compression,
		masterSeed:           make([]byte, MASTER_SEED_LEN),
		transformSeed:        make([]byte, TRANSFORM_SEED_LEN),
		transformRounds:      transformRounds,
		encryptionIV:         make([]byte, encryptionIVLength),
		innerRandomStreamKey: make([]byte, INNER_RANDOM_STREAM_KEY_LEN),
		streamStartBytes:     make([]byte, STREAM_START_BYTES_LEN),
		irsid:                irsid,
	}
}

func (h *header) randomize() {
	rand.Read(h.masterSeed)
	rand.Read(h.transformSeed)
	rand.Read(h.encryptionIV)
	rand.Read(h.streamStartBytes)
	rand.Read(h.innerRandomStreamKey[:])
}

func (h *header) read(stream *os.File) error {
	startOfHeader, err := stream.Seek(0, io.SeekCurrent)

	// Check filetype signature
	eq, err := util.ReadCompare(stream, FILE_SIGNATURE[:])
	if err != nil {
		return err
	}
	if !eq {
		return FileError(errors.New("Invalid file signature"))
	}

	// Check KeePass version signature
	eq, err = util.ReadCompare(stream, VERSION_SIGNATURE[:])
	if err != nil {
		return err
	}
	if !eq {
		return FileError(errors.New("Invalid or unsupported version signature"))
	}

	err = h.version.read(stream)
	if err != nil {
		return err
	}
	if h.version.major != 3 || h.version.minor != 1 {
		return FileError(errors.New("Only kdbx version 3.1 is supported"))
	}

	headerMap := make(map[headerCode][]byte)
	bufType := make([]byte, TLV_TYPE_LEN)
	bufLength := make([]byte, TLV_LENGTH_LEN)
	var (
		htype  headerCode
		length uint16
		value  []byte
	)
	for {
		err = util.ReadAssert(stream, bufType)
		if err != nil {
			return err
		}
		err = util.ReadAssert(stream, bufLength)
		if err != nil {
			return err
		}
		htype = headerCode(bufType[0])
		length = binary.LittleEndian.Uint16(bufLength)
		value = make([]byte, length)
		err = util.ReadAssert(stream, value)
		if err != nil {
			return err
		}
		if htype == EOH {
			break
		}
		if !validHeaderCode(htype) {
			log.Printf("WARNING: Skipping invalid header code: %d", htype)
		}
		headerMap[htype] = value
	}

	// Store hash
	endOfHeader, err := stream.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	if _, err = stream.Seek(startOfHeader, io.SeekStart); err != nil {
		return err
	}
	headerLength := endOfHeader - startOfHeader
	headerRaw := make([]byte, headerLength)
	stream.Read(headerRaw)
	h.hashOfRead = sha256.Sum256(headerRaw)

	// Parse header fields
	for _, h := range obligatoryFields {
		if _, present := headerMap[h]; !present {
			return FileError(fmt.Errorf("Missing header with code %d", h))
		}
	}

	if !bytes.Equal(headerMap[CipherID], AES_CIPHER_ID[:]) {
		return FileError(errors.New("Invalid or unsupported cipher"))
	}

	h.compression, err = getCompression(headerMap[CompressionFlag])
	if err != nil {
		return nil
	}

	h.masterSeed = headerMap[MasterSeed]
	h.transformSeed = headerMap[TransformSeed]
	h.transformRounds = binary.LittleEndian.Uint64(headerMap[TransformRounds])
	h.encryptionIV = headerMap[EncryptionIV]
	h.innerRandomStreamKey = headerMap[ProtectedStreamKey]
	h.streamStartBytes = headerMap[StreamStartBytes]

	irsid := binary.LittleEndian.Uint32(headerMap[InnerRandomStreamID])
	if !validIRSID(irsid) {
		return FileError(fmt.Errorf("Invalid Inner Random Stream ID: %d", irsid))
	}
	h.irsid = IRSID(irsid)

	return nil
}

// Write header to stream and return hash of bytes that were written
func (h *header) write(stream io.Writer) ([sha256.Size]byte, error) {
	buf := new(bytes.Buffer)
	err := util.WriteAssert(buf, FILE_SIGNATURE[:])
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	err = util.WriteAssert(buf, VERSION_SIGNATURE[:])
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	h.version.write(buf)
	if err != nil {
		return [sha256.Size]byte{}, err
	}

	compressionFlag := getCompressionFlag(h.compression)

	transformRoundsBuf := make([]byte, QWORD)
	binary.LittleEndian.PutUint64(transformRoundsBuf, h.transformRounds)
	irsBuf := make([]byte, DWORD)
	binary.LittleEndian.PutUint32(irsBuf, uint32(h.irsid))

	fields := []struct {
		id   headerCode
		data []byte
	}{
		{CipherID, AES_CIPHER_ID[:]},
		{CompressionFlag, compressionFlag},
		{MasterSeed, h.masterSeed},
		{TransformSeed, h.transformSeed},
		{TransformRounds, transformRoundsBuf},
		{EncryptionIV, h.encryptionIV},
		{ProtectedStreamKey, h.innerRandomStreamKey[:]},
		{StreamStartBytes, h.streamStartBytes},
		{InnerRandomStreamID, irsBuf},
		{EOH, EOH_DATA[:]},
	}
	for _, field := range fields {
		err := writeHeaderField(buf, field.id, field.data)
		if err != nil {
			return [sha256.Size]byte{}, err
		}
	}
	raw, _ := io.ReadAll(buf)
	hash := sha256.Sum256(raw)
	stream.Write(raw)
	return hash, nil
}

func writeHeaderField(stream io.Writer, id headerCode, data []byte) error {
	if len(data) > int(MAX_UINT16) {
		return fmt.Errorf("Header field exceeds maximum length: %d > %d", len(data), MAX_UINT16)
	}
	lengthBuf := make([]byte, WORD)
	binary.LittleEndian.PutUint16(lengthBuf, uint16(len(data)))
	err := util.WriteAssert(stream, []byte{byte(id)})
	if err != nil {
		return err
	}
	err = util.WriteAssert(stream, lengthBuf)
	if err != nil {
		return err
	}
	return util.WriteAssert(stream, data)
}

func getCompression(compressionFlag []byte) (bool, error) {
	switch flag := binary.LittleEndian.Uint32(compressionFlag); flag {
	case COMPRESSION_None:
		return false, nil
	case COMPRESSION_GZip:
		return true, nil
	default:
		return false, FileError(fmt.Errorf("Unknown compression flag: %d", flag))
	}
}

func getCompressionFlag(compression bool) []byte {
	buf := make([]byte, DWORD)
	flag := uint32(COMPRESSION_None)
	if compression {
		flag = COMPRESSION_GZip
	}
	binary.LittleEndian.PutUint32(buf, flag)
	return buf
}
