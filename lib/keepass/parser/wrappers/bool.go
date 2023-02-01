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

func (l *Bool) IsSet() bool {
	return l.isSet
}

func (l *Bool) Value() bool {
	return l.value
}

func (l *Bool) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
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
