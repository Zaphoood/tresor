package parser

import (
	"encoding/hex"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	GENERATOR            = "KeePass"
	PROTECTED_STREAM_KEY = "110f7392171a714fb4a7684c6f829a01d9eccb6a5089376eca23fe3bf4875a4c"
)

func TestParse(t *testing.T) {
	assert := assert.New(t)

	key, err := hex.DecodeString(PROTECTED_STREAM_KEY)
	if err != nil || len(key) != 32 {
		t.Fatal("Failed to decode hex string")
	}
	xmlFile, err := os.Open("../test/example_decrypted.xml")
	defer xmlFile.Close()
	if !assert.Nil(err) {
		return
	}

	byteValue, _ := ioutil.ReadAll(xmlFile)

	parsed, err := Parse(byteValue, *(*[32]byte)(key))
	if !assert.Nil(err) {
		return
	}

	assert.Equal(GENERATOR, parsed.Meta.Generator)
	// We know that the first entry in the database has password "Password"
	firstEntryUUID := parsed.Root.Groups[0].Entries[0].UUID
	assert.Equal("Password", parsed.Unlocked[firstEntryUUID].Strings[0].Value.Chardata)
}
