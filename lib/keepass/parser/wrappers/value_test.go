package wrappers

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/Zaphoood/tresor/lib/keepass/crypto"
	"github.com/stretchr/testify/assert"
)

func TestValue(t *testing.T) {
	assert := assert.New(t)

	v := Value{
		Inner:     "I love capybaras",
		Protected: true,
	}
	expectedXml := `<Value Protected="True">EXPgIU5fBPZ+HyP+4Dg+1A==</Value>`

	// Marshal without setting stream first
	_, err := xml.Marshal(&v)
	assert.NotNil(err)

	key := [32]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	salsa := crypto.NewSalsa20Stream(key)
	SetInnerRandomStream(salsa)

	document, err := xml.Marshal(&v)
	if !assert.Nil(err) {
		return
	}
	assert.Equal(string(document), expectedXml)
	assert.False(strings.Contains(string(document), v.Inner))

	salsa = crypto.NewSalsa20Stream(key)
	SetInnerRandomStream(salsa)

	vOut := Value{}
	err = xml.Unmarshal(document, &vOut)
	if !assert.Nil(err) {
		return
	}
	assert.Equal(v, vOut)
}
