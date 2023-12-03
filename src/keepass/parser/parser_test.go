package parser

import (
	"encoding/hex"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/Zaphoood/tresor/src/keepass/parser/wrappers"
	"github.com/stretchr/testify/assert"
)

const (
	GENERATOR            = "KeePass"
	PROTECTED_STREAM_KEY = "be3723cc9496ac62a51976df67314e68203140178c1aba143ce6c2441f1068f4"
)

func parseDecryptedExample(t *testing.T) *Document {
	key, err := hex.DecodeString(PROTECTED_STREAM_KEY)
	if err != nil || len(key) != 32 {
		t.Fatal("Failed to decode hex string")
	}
	xmlFile, err := os.Open("../test/example_decrypted.xml")
	defer xmlFile.Close()
	if err != nil {
		t.Fatal(err)
	}

	content, _ := ioutil.ReadAll(xmlFile)

	parsed, err := Parse(content, *(*[32]byte)(key))
	if err != nil {
		t.Fatal(err)
	}

	return parsed
}

func TestParse(t *testing.T) {
	assert := assert.New(t)

	parsed := parseDecryptedExample(t)

	assert.Equal(GENERATOR, parsed.Meta.Generator)
	// We know that the first entry in the database has password "Password"
	firstEntry := parsed.Root.Groups[0].Entries[0]
	password, err := firstEntry.Get("Password")
	if assert.Nil(err) {
		assert.Equal("Password", password.Inner)
	}

	_, err = firstEntry.Get("foo")
	assert.NotNil(err)

	fallback := "bar"
	result := firstEntry.TryGet("foo", fallback)
	assert.Equal(fallback, result)

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

	rootGroup := parsed.Root.Groups[0]
	assert.False(rootGroup.EnableAutoType.IsSet())

	assert.True(rootGroup.Groups[0].EnableAutoType.IsSet())
	assert.True(rootGroup.Groups[0].EnableAutoType.Value())

	assert.True(rootGroup.Groups[0].EnableSearching.IsSet())
	assert.False(rootGroup.Groups[0].EnableSearching.Value())

	uuid0 := "M0Gbdz4OmEaVH1j8pqgWFA==" // UUID of root group
	path, found := parsed.FindPath(uuid0)
	assert.True(found)
	assert.Equal([]string{uuid0}, path)

	uuid1 := "rrneGT70Vka3wdwglo3oDQ=="
	path, found = parsed.FindPath(uuid1)
	assert.True(found)
	assert.Equal(uuid1, path[len(path)-1])

	path = []string{"M0Gbdz4OmEaVH1j8pqgWFA==", "TLnGe1+SlES04aiZ9Sk0Kg==", "NZY6u4bWoUqJaIvckl3mLA=="}
	item, err := parsed.GetItem(path)
	entry, ok := item.(Entry)
	if assert.True(ok) {
		assert.Equal(entry.UUID, path[len(path)-1])
	}
}

func TestUpdateEntry(t *testing.T) {
	assert := assert.New(t)

	document := parseDecryptedExample(t)

	path := []string{"M0Gbdz4OmEaVH1j8pqgWFA==", "A/ntiXf2VEW3qSstTnhbcA=="}
	item, err := document.GetItem(path)
	if !assert.Nil(err) {
		return
	}
	entry, ok := item.(Entry)
	assert.True(ok, "Test entry is not of type Entry")

	newEntry := entry
	newEntry.Strings = []String{
		{
			Key: "foo", Value: wrappers.Value{
				Inner:     "bar",
				Protected: false,
			},
		},
	}
	document.UpdateEntry(newEntry)

	retrievedItem, err := document.GetItem(path)
	if !assert.Nil(err) {
		return
	}

	retrievedEntry, ok := retrievedItem.(Entry)
	assert.True(ok, "Retrieved entry is not of type Entry")

	assert.Equal(entry.UUID, retrievedEntry.UUID)
	assert.Equal(entry.IconID, retrievedEntry.IconID)
	assert.Equal(entry.BinaryRefs, retrievedEntry.BinaryRefs)
	assert.Equal(entry.History, retrievedEntry.History)

	assert.Equal(1, len(retrievedEntry.Strings))
	assert.Equal("foo", retrievedEntry.Strings[0].Key)
	assert.Equal("bar", retrievedEntry.Strings[0].Value.Inner)
}
