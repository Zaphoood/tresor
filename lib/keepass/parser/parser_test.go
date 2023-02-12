package parser

import (
	"encoding/hex"
	"io/ioutil"
	"os"
	"testing"
	"time"

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
	firstEntry := parsed.Root.Groups[0].Entries[0]
	firstEntryUUID := firstEntry.UUID
	assert.Equal("Password", parsed.Unlocked[firstEntryUUID].Strings[0].Value.Chardata)

	assert.Equal(2, len(firstEntry.BinaryRefs))
	for _, bref := range firstEntry.BinaryRefs {
		if bref.Reference.ID == 0 {
			assert.Equal("empty", bref.Key)
		} else if bref.Reference.ID == 1 {
			assert.Equal("myattachment.txt", bref.Key)
		} else {
			t.Errorf("Reference to binary with unexpected ID: %d", bref.Reference.ID)
		}
	}

	expectedBinaries := []struct {
		id    int
		value string
	}{
		{0, ""},
		{1, "This is an attachment\n"},
	}

	assert.Equal(len(expectedBinaries), len(parsed.Meta.Binaries))
	for _, b := range expectedBinaries {
		binary, err := parsed.GetBinary(b.id)
		if assert.Nil(err) {
			assert.Equal(b.value, string(binary))
		}
	}
	expectedDeletedItem := struct {
		uuid         string
		deletionTime time.Time
	}{"J1FUp3NO3ECuZtoZH54kHw==", time.Date(2023, time.February, 12, 22, 6, 16, 0, time.UTC)}
	assert.Equal(1, len(parsed.Root.DeletedObjects))
	assert.Equal(expectedDeletedItem.uuid, parsed.Root.DeletedObjects[0].UUID)
	assert.Equal(expectedDeletedItem.deletionTime, parsed.Root.DeletedObjects[0].DeletionTime)

	assert.False(parsed.Root.Groups[0].EnableAutoType.IsSet())

	assert.True(parsed.Root.Groups[0].Groups[0].EnableAutoType.IsSet())
	assert.True(parsed.Root.Groups[0].Groups[0].EnableAutoType.Value())

	assert.True(parsed.Root.Groups[0].Groups[0].EnableSearching.IsSet())
	assert.False(parsed.Root.Groups[0].Groups[0].EnableSearching.Value())
}
