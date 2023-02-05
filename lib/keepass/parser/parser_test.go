package parser

import (
	"io/ioutil"
	"log"
	"os"
	"testing"
)

const (
	GENERATOR = "KeePass"
)

func TestParse(t *testing.T) {
	xmlFile, err := os.Open("../test/example_decrypted.xml")
	if err != nil {
		t.Fatal(err)
	}

	defer xmlFile.Close()

	byteValue, _ := ioutil.ReadAll(xmlFile)

	parsed, err := Parse(byteValue)
	if err != nil {
		t.Fatal(err)
	}

	log.Printf("Generator: %s\n", parsed.Meta.Generator)
	log.Printf("HeaderHash: %s\n", parsed.Meta.HeaderHash)
	log.Printf("Recycle Bin changed: %s\n", parsed.Meta.RecycleBinChanged.String())

	if parsed.Meta.Generator != GENERATOR {
		t.Fatalf("For Generator: want %s, got '%s'", GENERATOR, parsed.Meta.Generator)
	}

	log.Printf("Groups (%d):\n", len(parsed.Root.Groups))
	for _, group := range parsed.Root.Groups {
		log.Printf(" * %s\n", group.Name)
		for _, entry := range group.Entries {
			val, err := entry.Get("Title")
			var title string
			if err == nil {
				title = val.Chardata
			} else {
				title = "(No title)"
			}
			log.Printf("   * %s (UUID: %s, history: %d)\n", title, entry.UUID, len(entry.History))
			for _, str := range entry.Strings {
				log.Printf("     * %s (Protected: %t): %s \n",
					str.Key, str.Value.IsProtected(), str.Value.Chardata)
			}
		}
	}

	path := []int{0, 3, 1}
	item, err := parsed.GetItem(path)
	if err != nil {
		log.Fatal(err)
	}
	group, ok := item.(Group)
	if ok {
		log.Printf("Groups at path %v:\n", path)
		for _, g := range group.Groups {
			log.Printf(" * %s", g.Name)
		}
	} else {
		log.Printf("Not listing path since there is not group at path %v\n", path)
	}
}
