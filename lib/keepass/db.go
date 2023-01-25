package keepass

import (
    "bytes"
    "errors"
    "encoding/binary"
    "fmt"
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
    InnerRandomstreamID
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
    InnerRandomstreamID,
}

func validHeaderCode(c headerCode) bool {
    return EOH <= c && c <= InnerRandomstreamID
}

var (
    FILE_SIGNATURE    [4]byte = [4]byte{ 0x03, 0xD9, 0xA2, 0x9A }
    VERSION_SIGNATURE [4]byte = [4]byte{ 0x67, 0xFB, 0x4B, 0xB5 }
)

type database struct {
    path     string
    content  string
    verMajor uint16
    verMinor uint16
}

type IRSID int

type headers struct {
    masterSeed         []byte
    transformSeed      []byte
    transformRounds    uint64
    encryptionIV       []byte
    protectedStreamKey []byte
    streamStartBytes   []byte
    irs                IRSID
}

func NewDatabase(path string) database {
    return database{path, "", 0, 0}
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
    log.Println("Parsing database")
    f, err := os.Open(d.path)
    defer f.Close()
    if err != nil { return err }

    // Check filetype signature
    eq, err := readCompare(f, FILE_SIGNATURE[:])
    if err != nil { return err }
    if !eq {
        return errors.New("Invalid file signature")
    }

    // Check KeePass version signature
    eq, err = readCompare(f, VERSION_SIGNATURE[:])
    if err != nil { return err }
    if !eq {
        return errors.New("Invalid or unsupported version signature")
    }

    // Read kdbx version
    bufMinor := make([]byte, VERSION_NUMBER_LEN)
    bufMajor := make([]byte, VERSION_NUMBER_LEN)
    read, err := f.Read(bufMinor)
    if err != nil { return err }
    if read != len(bufMinor) {
        return errors.New("File truncated")
    }
    read, err = f.Read(bufMajor)
    if err != nil { return err }
    if read != len(bufMajor) {
        return errors.New("File truncated")
    }
    d.verMinor = binary.LittleEndian.Uint16(bufMinor)
    d.verMajor = binary.LittleEndian.Uint16(bufMajor)
    log.Printf("Version is %d.%d", d.verMajor, d.verMinor)

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
        if err != nil { return err }
        if read != len(bufType) {
            return errors.New("File truncated")
        }
        read, err = f.Read(bufLength)
        if err != nil { return err }
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

    // Read remaining file content


    return nil
}

// Read len(b) bytes from f and compare with b
func readCompare(f *os.File, b []byte) (bool, error) {
    buf := make([]byte, len(b))
    _, err := f.Read(buf)
    if err != nil { return false, err }
    return bytes.Equal(buf, b), nil
}

