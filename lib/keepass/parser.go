package keepass

import (
	"encoding/xml"
	"fmt"
	"time"
)

type LiteralBool struct {
	isSet bool
	value bool
}

func (l *LiteralBool) IsSet() bool {
	return l.isSet
}

func (l *LiteralBool) Value() bool {
	return l.value
}

func (l *LiteralBool) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	l.isSet = false
	var charData xml.CharData
	charData = nil
	end := false
	for !end {
		token, err := d.Token()
		if err != nil {
			return err
		}
		switch t := token.(type) {
		case xml.CharData:
			charData = t.Copy()
		case xml.EndElement:
			end = true
		}
	}
	if charData == nil {
		return nil
	}
	switch string(charData) {
	case "True":
		l.isSet = true
		l.value = true
		return nil
	case "False":
		l.isSet = true
		l.value = false
		return nil
	default:
		return fmt.Errorf("Failed to parse element '%s' as literal bool. Want 'True' or 'False', got %s", start.Name.Local, charData)
	}
}

type Parsed struct {
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
	RecycleBinEnabled          LiteralBool
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
	ProtectTitle    LiteralBool
	ProtectUserName LiteralBool
	ProtectPassword LiteralBool
	ProtectURL      LiteralBool
	ProtectNotes    LiteralBool
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
	IsExpanded LiteralBool
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
	Expires              LiteralBool
	UsageCount           int
	LocationChanged      time.Time
}

type AutoType struct {
	Enabled                 LiteralBool
	DataTransferObfuscation int
	Association             Association
}

type Association struct {
	Window            string
	KeystrokeSequence string
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

func Parse(b []byte) (*Parsed, error) {
	p := Parsed{}
	err := xml.Unmarshal(b, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
