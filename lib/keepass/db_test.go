package keepass

import (
    "fmt"
    "testing"
)

const (
    XML_HEADER = "<?xml version=\"1.0\" encoding=\"utf-8\" standalone=\"yes\"?>"
)

func TestFileNotExist(t *testing.T) {
    d := NewDatabase("/this/path/does/not/exist.kdbx")
    err := d.Load("password")
    if err == nil {
        t.Error("Want error for non-existent path, got nil")
    }
}

func TestLoadDb(t *testing.T) {
    d := NewDatabase("../../example/example.kdbx")
    err := d.Load("password")
    if err != nil {
        t.Fatal(err)
    }

    expectedMajor := uint16(3)
    expectedMinor := uint16(1)
    if major, minor := d.Version(); major != expectedMajor || minor != expectedMinor {
      t.Errorf("Want version to be %d.%d but got %d.%d", expectedMajor, expectedMinor, major, minor)
    }

    //if !strings.HasPrefix(d.Content(), XML_HEADER) {
    //    t.Error(fmt.Sprintf("Missing XML header, got:\n%s", d.Content()))
    //}
    fmt.Println("Headers:")
    fmt.Printf("masterSeed: %x\n", d.headers.masterSeed)
    fmt.Printf("transformSeed: %x\n", d.headers.transformSeed)
    fmt.Printf("transformRounds: %d\n", d.headers.transformRounds)
    fmt.Printf("encryptionIV: %x\n", d.headers.encryptionIV)
    fmt.Printf("protectedStreamKey: %x\n", d.headers.protectedStreamKey)
    fmt.Printf("streamStartBytes: %x\n", d.headers.streamStartBytes)
    fmt.Printf("irs: %d\n", d.headers.irs)
    fmt.Printf("Content:\n%s", d.Content())
}

func TestInvalidFileSignature(t *testing.T) {
    d := NewDatabase("../../example/example_invalid_file_signature.kdbx")
    err := d.Load("password")
    if err == nil {
        t.Fatal("Want error for file with invalid file signature, got nil")
    }
}

func TestInvalidVersionSignature(t *testing.T) {
    d := NewDatabase("../../example/example_invalid_version_signature.kdbx")
    err := d.Load("password")
    if err == nil {
        t.Fatal("Want error for file with invalid version signature, got nil")
    }
}

func TestInvalidCipherID(t *testing.T) {
    d := NewDatabase("../../example/example_invalid_cipher_id.kdbx")
    err := d.Load("password")
    if err == nil {
        t.Fatal("Want error for file with invalid cipher id, got nil")
    }
}

func TestCompressed(t *testing.T) {
    // Compression is not implemented yet, so we want to return an error for compressed databases
    d := NewDatabase("../../example/example_compressed.kdbx")
    err := d.Load("password")
    if err == nil {
        t.Fatal("Want error for compressed database, got nil")
    }
}
