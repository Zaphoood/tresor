package wrappers

import (
	"encoding/xml"
	"fmt"
	"strings"
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
	switch strings.ToLower(string(charData)) {
	case "true":
		b.isSet = true
		b.value = true
		return nil
	case "false":
		b.isSet = true
		b.value = false
		return nil
	case "null":
		b.isSet = false
		return nil
	default:
		return fmt.Errorf("Failed to unmarshal element '%s' as literal bool. Want 'True' or 'False', got '%s'", start.Name.Local, charData)
	}
}

func (b *Bool) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	var str string
	switch {
	case !b.isSet:
		str = "null"
	case b.value:
		str = "True"
	case !b.value:
		str = "False"
	}
	return e.EncodeElement(str, start)
}
