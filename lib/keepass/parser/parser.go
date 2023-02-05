package parser

import (
	"encoding/xml"
	"errors"
	"fmt"
	"time"

	"github.com/Zaphoood/tresor/lib/keepass/parser/wrappers"
)

type Document struct {
	XMLName xml.Name `xml:"KeePassFile"`
	Meta    Meta
	Root    Root
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
	RecycleBinChanged          time.Time
	EntryTemplatesGroup        string
	EntryTemplatesGroupChanged time.Time
	HistoryMaxItems            int
	HistoryMaxSize             int
	LastSelectedGroup          string
	LastTopVisibleGroup        string
	//Binaries
	//CustomData
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
	XMLName xml.Name `xml:"Root"`
	Groups  []Group  `xml:"Group"`
	//DeletedObjects
}

type Item interface{}

type Group struct {
	XMLName xml.Name `xml:"Group"`
	Groups  []Group  `xml:"Group"`
	Entries []Entry  `xml:"Entry"`

	UUID       string
	Name       string
	IconID     int
	Times      Times
	IsExpanded wrappers.Bool
	//DefaultAutoTypeSequence // string?
	// EnableAutoType // bool?
	//EnableSearching // bool?
	LastTopVisibleEntry string
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
	AutoType        AutoType
	Strings         []String `xml:"String"`
	History         []Entry  `xml:"History>Entry"`
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

type AutoType struct {
	Enabled                 wrappers.Bool
	DataTransferObfuscation int
	Association             Association
}

type Association struct {
	Window            string
	KeystrokeSequence string
}

func (e *Entry) Get(key string) (string, error) {
	for _, str := range e.Strings {
		if str.Key == key {
			return str.Value.Chardata, nil
		}
	}
	return "", fmt.Errorf("No such key: %s", key)
}

func (e *Entry) TryGet(key, fallback string) string {
	result, err := e.Get(key)
	if err != nil {
		return fallback
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

func Parse(b []byte) (*Document, error) {
	p := Document{}
	err := xml.Unmarshal(b, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
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

type PathOutOfRange error
