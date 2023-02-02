package wrappers

import (
	"encoding/xml"
	"fmt"
)

// Bool represents a tag that contains either "True" or "False" as its chardata
type Bool struct {
	isSet bool
	value bool
}

func (b *Bool) IsSet() bool {
	return b.isSet
}

func (b *Bool) Value() bool {
	return b.value
}

func (b *Bool) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	b.isSet = false
	var charData xml.CharData
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
		return fmt.Errorf("Failed to unmarshal element '%s' as literal bool: empty element", start.Name.Local)
	}
	switch string(charData) {
	case "True":
		b.isSet = true
		b.value = true
		return nil
	case "False":
		b.isSet = true
		b.value = false
		return nil
	default:
		return fmt.Errorf("Failed to unmarshal element '%s' as literal bool. Want 'True' or 'False', got '%s'", start.Name.Local, charData)
	}
}
