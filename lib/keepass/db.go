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
    VERSION_NUMBER_LEN  = 2
)

var (
    FILE_SIGNATURE    [4]byte = [4]byte{ 3,   217, 162, 154 }
    VERSION_SIGNATURE [4]byte = [4]byte{ 103, 251,  75, 181 }
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

    // Parse headers

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
