package wrappers

import (
	"encoding/base64"
	"encoding/xml"
	"errors"
	"strings"

	"github.com/Zaphoood/tresor/lib/keepass/crypto"
)

var stream crypto.Stream

func SetInnerRandomStream(s crypto.Stream) {
	stream = s
}

type Value struct {
	XMLName   xml.Name `xml:"Value"`
	Inner     string
	Protected bool
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
