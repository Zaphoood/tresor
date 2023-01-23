package keepass

import (
    "errors"
    "fmt"
    "os"
)

type database struct {
    path     string
    password string
    content  string
}

func NewDatabase(path, password string) database {
    return database{path, password, ""}
}

func (d database) Content() string {
    return d.content
}

func (d *database) Load() error {
    // Check if file exists
    _, err := os.Stat(d.path)
    if err != nil {
        if os.IsNotExist(err) {
            return errors.New(fmt.Sprintf("File %s does not exist", d.path))
        }
    }
    d.content = "Hello :)"
    return nil
}
