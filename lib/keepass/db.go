package keepass

import (
    "fmt"
    "time"
    "math/rand"
    "errors"
)

func LoadDB(path string) (string, error) {
    // Mock database loading -- act like it failed 50% of the time
    time.Sleep(500 * time.Millisecond)
    rand.Seed(time.Now().UnixNano())
    if (rand.Int() % 2) == 1 {
        return "", errors.New(fmt.Sprintf("Failed to load DB %s", path))
    }
    return fmt.Sprintf("Content of database %s:\nuser=johndoe\npassword=foobar", path), nil
}
