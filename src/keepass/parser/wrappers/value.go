package wrappers

import (
	"encoding/base64"
	"encoding/xml"
	"errors"
	"strings"

	"github.com/Zaphoood/tresor/src/keepass/crypto"
)

var stream crypto.Stream

func SetInnerRandomStream(s crypto.Stream) {
	stream = s
}

type Value struct {
	XMLName   xml.Name `xml:"Value"`
	Inner     string   `xml:",chardata"`
	Protected bool     `xml:",attr"`
}

func (v *Value) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if attr.Name.Local == "Protected" && strings.ToLower(attr.Value) == "true" {
			v.Protected = true
		}
	}
	chardata := ""
	end := false
	for !end {
		token, err := d.Token()
		if err != nil {
			return err
		}
		switch t := token.(type) {
		case xml.CharData:
			chardata = string(t.Copy())
		case xml.EndElement:
			end = true
		}
	}
	if v.Protected {
		decoded, err := base64.StdEncoding.DecodeString(chardata)
		if err != nil {
			return err
		}
		if stream == nil {
			return errors.New("Error while unmarshalling protected Value: stream is nil")
		}
		decrypted, err := stream.Decrypt(decoded)
		if err != nil {
			return err
		}
		v.Inner = string(decrypted)
	} else {
		v.Inner = chardata
	}
	return nil
}

func (v *Value) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if !v.Protected {
		return e.EncodeElement(v.Inner, start)
	}

	start.Attr = []xml.Attr{
		{
			Name:  xml.Name{Local: "Protected"},
			Value: "True",
		},
	}

	if stream == nil {
		return errors.New("Error while unmarshalling protected Value: stream is nil")
	}
	encrypted, err := stream.Encrypt([]byte(v.Inner))
	if err != nil {
		return err
	}
	encoded := base64.StdEncoding.EncodeToString(encrypted)

	return e.EncodeElement(encoded, start)
}
