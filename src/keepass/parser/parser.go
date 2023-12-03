package parser

import (
	"encoding/xml"
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

type Group struct {
	XMLName                 xml.Name `xml:"Group"`
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
	Entries                 []Entry `xml:"Entry"`
	Groups                  []Group `xml:"Group"`
}

type Entry struct {
	XMLName         xml.Name `xml:"Entry"`
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

	marshalled, err := xml.MarshalIndent(d, "", "\t")
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(xml.Header)+len(marshalled))
	out = append(out, xml.Header...)
	out = append(out, marshalled...)

	return out, nil
}
