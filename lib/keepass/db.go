package keepass

import (
    "bytes"
    "errors"
    "fmt"
    "log"
    "os"
)

var (
    FILE_SIGNATURE    [4]byte = [4]byte{ 3,   217, 162, 154 }
    VERSION_SIGNATURE [4]byte = [4]byte{ 103, 251,  75, 181 }
)

type database struct {
    path     string
    content  string
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
    return database{path, ""}
}

func (d database) Content() string {
    return d.content
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
