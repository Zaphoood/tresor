package parser

import (
	"encoding/xml"
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
// each level choosing the group with the current index, until the end of the path is reached.
// An empty path will result in the top-level groups being returned
func (d *Document) ListPath(path []int) ([]Group, []Entry, error) {
	entries := []Entry{}
	current := &d.Root.Groups
	for i := 0; i < len(path); i++ {
		if path[i] >= len(*current) {
			return []Group{}, []Entry{}, fmt.Errorf("Path entry at position %d is out of range: %d >= %d", i, path[i], len(*current))
		}
		if i == len(path)-1 {
			entries = (*current)[path[i]].Entries
		}
		current = &(*current)[path[i]].Groups
	}
	return *current, entries, nil
}

func (d *Document) GetItem(path []int) (interface{}, error) {
	groups, entries, err := d.ListPath(path[:len(path)-1])
	if err != nil {
		return nil, err
	}
	index := path[len(path)-1]
	if index < len(groups) {
		return groups[index], nil
	} else if index < len(groups)+len(entries) {
		return entries[index-len(groups)], nil
	} else {
		return nil, fmt.Errorf("Item index out of range: %d", index)
	}
}
