package keepass

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
	xmlFile, err := os.Open("test/example_decrypted.xml")
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
			title, err := entry.Get("Title")
			if err != nil {
				title = "(No title)"
			}
			log.Printf("   * %s (UUID: %s, history: %d)\n", title, entry.UUID, len(entry.History))
			for _, str := range entry.Strings {
				log.Printf("     * %s (Protected: %t): %s \n",
					str.Key, str.Value.IsProtected(), str.Value.Chardata)
			}
			//for _, str := range entry.Strings {
			//	log.Printf("     * %s: %s\n", str.Key, str.Value)
			//}
		}
	}
}
