package parser

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"time"

	"github.com/Zaphoood/tresor/lib/keepass/crypto"
	"github.com/Zaphoood/tresor/lib/keepass/parser/wrappers"
	"github.com/antchfx/xmlquery"
)

type Document struct {
	XMLName xml.Name `xml:"KeePassFile"`
	Meta    Meta
	Root    Root
	// Values with inner stream encryption are stored here after decrypting
	Unlocked map[string]Entry `xml:"-"`
}

func NewDocument() *Document {
	return &Document{}
}

type Meta struct {
	XMLName                    xml.Name `xml:"Meta"`
	Generator                  string
	HeaderHash                 string
	DatabaseName               string
	DatabaseNameChanged        time.Time
	DatabaseDescription        string
	DatabaseDescriptionChanged time.Time
	DefaultUserName            string
	DefaultUserNameChanged     time.Time
	MaintenanceHistoryDays     int
	Color                      string
	MasterKeyChanged           time.Time
	MasterKeyChangeRec         int
	MasterKeyChangeForce       int
	MemoryProtection           MemoryProtection
	RecycleBinEnabled          wrappers.Bool
	RecycleBinUUID             string
	RecycleBinChanged          time.Time
	EntryTemplatesGroup        string
	EntryTemplatesGroupChanged time.Time
	HistoryMaxItems            int
	HistoryMaxSize             int
	LastSelectedGroup          string
	LastTopVisibleGroup        string
	Binaries                   []Binary `xml:"Binaries>Binary"`
	CustomData                 CustomData
}

type CustomData struct {
	Inner string `xml:",innerxml"`
}

type Binary struct {
	XMLName  xml.Name `xml:"Binary"`
	ID       int      `xml:"ID,attr"`
	Chardata string   `xml:",chardata"`
}

type MemoryProtection struct {
	XMLName         xml.Name `xml:"MemoryProtection"`
	ProtectTitle    wrappers.Bool
	ProtectUserName wrappers.Bool
	ProtectPassword wrappers.Bool
	ProtectURL      wrappers.Bool
	ProtectNotes    wrappers.Bool
}

type Root struct {
	XMLName        xml.Name        `xml:"Root"`
	Groups         []Group         `xml:"Group"`
	DeletedObjects []DeletedObject `xml:"DeletedObjects>DeletedObject"`
}

type DeletedObject struct {
	UUID         string
	DeletionTime time.Time
}

type Item interface{}

type Group struct {
	XMLName xml.Name `xml:"Group"`
	Groups  []Group  `xml:"Group"`
	Entries []Entry  `xml:"Entry"`

	UUID                    string
	Name                    string
	IconID                  int
	Times                   Times
	IsExpanded              wrappers.Bool
	DefaultAutoTypeSequence string
	EnableAutoType          wrappers.Bool
	EnableSearching         wrappers.Bool
	LastTopVisibleEntry     string
}

func (g *Group) At(index int) (Item, error) {
	if index < 0 {
		return nil, fmt.Errorf("Negative index: %d", index)
	}
	if index < len(g.Groups) {
		return g.Groups[index], nil
	}
	if index-len(g.Groups) < len(g.Entries) {
		return g.Entries[index-len(g.Groups)], nil
	}
	return nil, fmt.Errorf("Index out of range for group '%s': %d", g.Name, index)
}

type Entry struct {
	XMLName xml.Name `xml:"Entry"`

	UUID            string
	IconID          int
	ForegroundColor string
	BackgroundColor string
	OverrideURL     string
	Tags            string
	Times           Times
	Strings         []String          `xml:"String"`
	BinaryRefs      []BinaryReference `xml:"Binary"`
	AutoType        AutoType
	History         []Entry `xml:"History>Entry"`
}

type Times struct {
	CreationTime         time.Time
	LastModificationTime time.Time
	LastAccessTime       time.Time
	ExpiryTime           time.Time
	Expires              wrappers.Bool
	UsageCount           int
	LocationChanged      time.Time
}

type BinaryReference struct {
	Key       string
	Reference BinaryReferenceValue `xml:"Value"`
}

type BinaryReferenceValue struct {
	ID int `xml:"Ref,attr"`
}

type AutoType struct {
	Enabled                 wrappers.Bool
	DataTransferObfuscation int
	Association             Association
}

type Association struct {
	Window            string
	KeystrokeSequence string
}

func (e *Entry) Get(key string) (Value, error) {
	for _, field := range e.Strings {
		if field.Key == key {
			return field.Value, nil
		}
	}
	return Value{}, fmt.Errorf("No such key: %s", key)
}

func (e *Entry) TryGet(key, fallback string) Value {
	result, err := e.Get(key)
	if err != nil {
		return Value{Chardata: fallback}
	}
	return result
}

type String struct {
	XMLName xml.Name `xml:"String"`
	Key     string
	Value   Value
}

type Value struct {
	XMLName   xml.Name `xml:"Value"`
	Chardata  string   `xml:",chardata"`
	Protected string   `xml:"Protected,attr"`
}

func (v *Value) IsProtected() bool {
	return v.Protected == "True"
}

func Parse(b []byte, key [32]byte) (*Document, error) {
	p := NewDocument()
	p.Unlock(b, key)

	err := xml.Unmarshal(b, &p)
	if err != nil {
		return nil, err
	}
	return p, nil
}

type field struct {
	key   string
	value string
}

// Unlock scans the XML for protected Values and stores their decrypted values in d.Unlocked
func (d *Document) Unlock(b []byte, key [32]byte) error {
	unlocked := make(map[string]Entry)
	salsa := crypto.NewSalsa20Stream(key)

	doc, err := xmlquery.Parse(bytes.NewReader(b))
	if err != nil {
		return err
	}

	for _, entryElement := range xmlquery.Find(doc, `//Group/Entry`) {
		currentEntry, err := unlockEntry(entryElement, salsa)
		if err != nil {
			return err
		}
		if history := xmlquery.FindOne(entryElement, "/History"); history != nil {
			for _, entryElementH := range xmlquery.Find(history, `/Entry`) {
				currentEntryH, err := unlockEntry(entryElementH, salsa)
				if err != nil {
					return err
				}
				currentEntry.History = append(currentEntry.History, currentEntryH)
			}
		}
		unlocked[currentEntry.UUID] = currentEntry
	}
	d.Unlocked = unlocked

	return nil
}

func unlockEntry(entryElement *xmlquery.Node, salsa *crypto.Salsa20Stream) (Entry, error) {
	entry := Entry{}
	uuidElement := xmlquery.FindOne(entryElement, "//UUID")
	if uuidElement == nil {
		return Entry{}, fmt.Errorf("<Entry> element without a <UUID> element:\n%s\n", entryElement.OutputXML(true))
	}
	entry.UUID = uuidElement.InnerText()

	protectedValues, err := listProtected(entryElement)
	if err != nil {
		return Entry{}, err
	}
	for _, protected := range protectedValues {
		decoded, err := base64.StdEncoding.DecodeString(protected.value)
		if err != nil {
			return Entry{}, err
		}
		decrypted := salsa.Decrypt(decoded)
		entry.Strings = append(entry.Strings, String{Key: protected.key, Value: Value{Chardata: string(decrypted)}})
	}
	return entry, nil
}

// listProtected returns all <Value> nodes with the Protected attribute set to 'True' as a list of fields
func listProtected(node *xmlquery.Node) ([]field, error) {
	fields := make([]field, 0)
	for _, protected := range xmlquery.Find(node, "/String/Value[@Protected='True']") {
		if keyNode := xmlquery.FindOne(protected.Parent, "//Key"); keyNode != nil {
			fields = append(fields, field{key: keyNode.InnerText(), value: protected.InnerText()})
		} else {
			return nil, fmt.Errorf("String element without a <Key> element:\n%s\n", protected.OutputXML(true))
		}
	}
	return fields, nil
}

func (d *Document) GetUnlocked(uuid, key string) (string, error) {
	if entry, exists := d.Unlocked[uuid]; exists {
		value, err := entry.Get(key)
		if err == nil {
			return value.Chardata, nil
		}
		return "", err
	} else {
		return "", fmt.Errorf("No entry with UUID '%s' found in Unlocked", uuid)
	}
}

// ListPath returns subgroups and entries of a group specified by an array of indices. The document is traversed,
// at each level choosing the group with the current index, until the end of the path is reached.
// For an empty path the function will return the top-level groups (which is just one group for most KeePass files)
func (d *Document) GetItem(path []int) (Item, error) {
	current := Group{Groups: d.Root.Groups}

	for i := 0; i < len(path); i++ {
		next, err := current.At(path[i])
		if err != nil {
			return nil, PathOutOfRange(fmt.Errorf("Invalid path entry at position %d: %s", i, err))
		}
		switch next := next.(type) {
		case Group:
			current = next
		case Item:
			if i == len(path)-1 {
				return next, nil
			}
			return nil, errors.New("Got Entry for non-final step in path")
		default:
			return nil, errors.New("Expected Group or Entry from Group.At()")
		}
	}
	return current, nil
}

func (d *Document) GetBinary(id int) ([]byte, error) {
	for _, binary := range d.Meta.Binaries {
		if binary.ID == id {
			decoded, err := base64.StdEncoding.DecodeString(binary.Chardata)
			if err != nil {
				return []byte{}, err
			}
			return decoded, nil
		}
	}
	return []byte{}, fmt.Errorf("No binary with ID: %d", id)
}

type PathOutOfRange error
