package keepass

import (
    //"log"
    "fmt"
    //"strings"
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
