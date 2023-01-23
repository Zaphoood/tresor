package keepass

import (
    "fmt"
    "strings"
    "testing"
)

const (
    XML_HEADER = "<?xml version=\"1.0\" encoding=\"utf-8\" standalone=\"yes\"?>"
)

func TestFileNotExist(t *testing.T) {
    d := NewDatabase("/this/path/does/not/exist.kdbx", "password")
    err := d.Load()
    if err == nil {
        t.Error("Want error for non-existent path, got nil")
    }
}

func TestLoadDb(t *testing.T) {
    d := NewDatabase("../../example/example.kdbx", "password")
    err := d.Load()
    if err != nil {
        t.Fatal(err)
    }
    if !strings.HasPrefix(d.content, XML_HEADER) {
        t.Error(fmt.Sprintf("Missing XML header, got:\n%s", d.Content()))
    }
}
