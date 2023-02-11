package database

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
)

const (
	TLV_TYPE_LEN   = 1
	TLV_LENGTH_LEN = 2
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

type header struct {
	gzipCompression    bool
	masterSeed         []byte
	transformSeed      []byte
	transformRounds    uint64
	encryptionIV       []byte
	protectedStreamKey [32]byte
	streamStartBytes   []byte
	irs                IRSID
}

func (h *header) read(stream io.Reader) error {
	headerMap := make(map[headerCode][]byte)
	bufType := make([]byte, TLV_TYPE_LEN)
	bufLength := make([]byte, TLV_LENGTH_LEN)
	var (
		htype  headerCode
		length uint16
		value  []byte
	)
	for {
		read, err := stream.Read(bufType)
		if err != nil {
			return err
		}
		if read != len(bufType) {
			return FileError{errors.New("File truncated")}
		}
		read, err = stream.Read(bufLength)
		if err != nil {
			return err
		}
		if read != len(bufLength) {
			return FileError{errors.New("File truncated")}
		}
		htype = headerCode(bufType[0])
		length = binary.LittleEndian.Uint16(bufLength)
		value = make([]byte, length)
		read, err = stream.Read(value)
		if htype == EOH {
			break
		}
		if !validHeaderCode(htype) {
			log.Printf("WARNING: Skipping invalid header code: %d", htype)
		}
		headerMap[htype] = value
	}

	// Parse headers
	for _, h := range obligatoryFields {
		if _, present := headerMap[h]; !present {
			return FileError{fmt.Errorf("Missing header with code %d", h)}
		}
	}

	if !bytes.Equal(headerMap[CipherID], AES_CIPHER_ID[:]) {
		return FileError{errors.New("Invalid or unsupported cipher")}
	}

	switch flag := binary.LittleEndian.Uint32(headerMap[CompressionFlag]); flag {
	case COMPRESSION_None:
		h.gzipCompression = false
	case COMPRESSION_GZip:
		h.gzipCompression = true
	default:
		return FileError{fmt.Errorf("Unknown compression flag: %d", flag)}
	}

	h.masterSeed = headerMap[MasterSeed]
	h.transformSeed = headerMap[TransformSeed]
	h.transformRounds = binary.LittleEndian.Uint64(headerMap[TransformRounds])
	h.encryptionIV = headerMap[EncryptionIV]
	h.protectedStreamKey = sha256.Sum256(headerMap[ProtectedStreamKey])
	h.streamStartBytes = headerMap[StreamStartBytes]

	irsid := binary.LittleEndian.Uint32(headerMap[InnerRandomStreamID])
	if !validIRSID(irsid) {
		return FileError{fmt.Errorf("Invalid Inner Random Stream ID: %d", irsid)}
	}
	h.irs = IRSID(irsid)

	return nil
}
