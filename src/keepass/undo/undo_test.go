package undo

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/Zaphoood/tresor/src/keepass/parser"
	"github.com/stretchr/testify/assert"
)

const (
	PROTECTED_STREAM_KEY = "be3723cc9496ac62a51976df67314e68203140178c1aba143ce6c2441f1068f4"
)

func parseDecryptedExample(t *testing.T) *parser.Document {
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

	parsed, err := parser.Parse(content, *(*[32]byte)(key))
	if err != nil {
		t.Fatal(err)
	}

	return parsed
}

func assertGetEntry(d *parser.Document, path []string) parser.Entry {
	item, err := d.GetItem(path)
	if err != nil {
		panic(err)
	}
	entry, ok := item.(parser.Entry)
	if !ok {
		panic(fmt.Sprintf("Test entry is not of type Entry (path %#v)", path))
	}
	return entry
}

type returnSentinel struct{}

func TestUpdateEntry(t *testing.T) {
	assert := assert.New(t)

	document := parseDecryptedExample(t)
	u := NewUndoManager[parser.Document]()

	path := []string{"M0Gbdz4OmEaVH1j8pqgWFA==", "A/ntiXf2VEW3qSstTnhbcA=="}
	entry := assertGetEntry(document, path)
	titleField, err := entry.Get("Title")
	if err != nil {
		fmt.Println(err)
		return
	}
	originalTitle := titleField.Inner
	newEntry := entry
	newTitle := "foo"
	newEntry.UpdateField("Title", newTitle)

	description := "Description"
	result := u.Do(document, NewUpdateEntryAction(newEntry, entry, returnSentinel{}, description))
	assert.Equal(result, returnSentinel{})

	entry2 := assertGetEntry(document, path)
	assert.Equal(newTitle, entry2.TryGet("Title", "(Failed to get field"))

	result, err = u.Undo(document)
	if assert.Nil(err) {
		assert.Equal(result, returnSentinel{})
	}

	entry3 := assertGetEntry(document, path)
	assert.Equal(originalTitle, entry3.TryGet("Title", "(Failed to get field"))

	result, err = u.Redo(document)
	if assert.Nil(err) {
		assert.Equal(result, returnSentinel{})
	}

	entry4 := assertGetEntry(document, path)
	assert.Equal(newTitle, entry4.TryGet("Title", "(Failed to get field"))

	assert.True(true)
}
