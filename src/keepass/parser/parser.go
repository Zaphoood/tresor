package parser

import (
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"time"

	"github.com/Zaphoood/tresor/src/keepass/crypto"
	"github.com/Zaphoood/tresor/src/keepass/parser/wrappers"
)

type Document struct {
	XMLName xml.Name `xml:"KeePassFile"`
	Meta    Meta
	Root    Root
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

	UUID                    string
	Name                    string
	Notes                   string
	IconID                  int
	Times                   Times
	IsExpanded              wrappers.Bool
	DefaultAutoTypeSequence string
	EnableAutoType          wrappers.Bool
	EnableSearching         wrappers.Bool
	LastTopVisibleEntry     string

	Entries []Entry `xml:"Entry"`
	Groups  []Group `xml:"Group"`
}

func (g *Group) Get(uuid string) (Item, error) {
	for _, group := range g.Groups {
		if group.UUID == uuid {
			return group, nil
		}
	}
	for _, entry := range g.Entries {
		if entry.UUID == uuid {
			return entry, nil
		}
	}
	return nil, fmt.Errorf("Group '%s' has no item with UUID '%s'", g.Name, uuid)
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
	// History must be pointer to slice in order for omitempty to work for nested elements
	History *[]Entry `xml:"History>Entry,omitempty"`
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
	Association             *Association `xml:",omitempty"`
}

type Association struct {
	Window            string
	KeystrokeSequence string
}

func (e *Entry) Get(key string) (wrappers.Value, error) {
	for _, field := range e.Strings {
		if field.Key == key {
			return field.Value, nil
		}
	}
	return wrappers.Value{}, fmt.Errorf("No such key: %s", key)
}

// TryGet returns the value for the given key if it exists, fallback otherwise
func (e *Entry) TryGet(key, fallback string) string {
	result, err := e.Get(key)
	if err != nil {
		return fallback
	}
	return result.Inner
}

type String struct {
	XMLName xml.Name `xml:"String"`
	Key     string
	Value   wrappers.Value
}

func Parse(b []byte, innerRandomStreamKey [32]byte) (*Document, error) {
	p := NewDocument()

	salsa := crypto.NewSalsa20Stream(innerRandomStreamKey)
	wrappers.SetInnerRandomStream(salsa)
	err := xml.Unmarshal(b, &p)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func Unparse(d *Document, key [32]byte) ([]byte, error) {
	salsa := crypto.NewSalsa20Stream(key)
	wrappers.SetInnerRandomStream(salsa)

	out, err := xml.MarshalIndent(d, "", "\t")
	if err != nil {
		return nil, err
	}
	unparsed := make([]byte, 0, len(xml.Header)+1+len(out))
	unparsed = append(unparsed, xml.Header...)
	unparsed = append(unparsed, byte('\n'))
	unparsed = append(unparsed, out...)

	return out, nil
}

type field struct {
	key   string
	value string
}

// GetItem returns a group or an item specified by a path of UUIDs. The document is traversed,
// at each level choosing the group with UUID at the current index, until the end of the path is reached.
// The last UUID may be that of an item.
// For an empty path the function will return the top-level groups (which is just one group for most KeePass files)
func (d *Document) GetItem(path []string) (Item, error) {
	current := Group{Groups: d.Root.Groups}

	for i := 0; i < len(path); i++ {
		next, err := current.Get(path[i])
		if err != nil {
			return nil, PathNotFound(fmt.Errorf("Invalid path entry at position %d: %s", i, err))
		}
		switch next := next.(type) {
		case Group:
			current = next
		case Entry:
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

// FindPath returns the relative path to a subgroup with the given UUID if it exists,
// and a bool indicating wether the UUID was found.
func (d *Document) FindPath(uuid string) ([]string, bool) {
	return findPathInGroups(uuid, d.Root.Groups)
}

func findPathInGroups(uuid string, groups []Group) ([]string, bool) {
	for _, group := range groups {
		if group.UUID == uuid {
			return []string{group.UUID}, true
		}
		subpath, found := findPathInGroups(uuid, group.Groups)
		if found {
			return append([]string{group.UUID}, subpath...), true
		}
	}
	return nil, false
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

type PathNotFound error
