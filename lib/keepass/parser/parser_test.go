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
	PROTECTED_STREAM_KEY = "be3723cc9496ac62a51976df67314e68203140178c1aba143ce6c2441f1068f4"
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
