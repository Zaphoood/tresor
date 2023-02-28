package wrappers

import (
	"encoding/xml"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBool(t *testing.T) {
	type root struct {
		XMLName xml.Name `xml:"Root"`
		Bool    Bool     `xml:"MyBool"`
	}
	template := "<Root><MyBool>%s</MyBool></Root>"
	cases := []struct {
		input         string
		expectError   bool
		expectedIsSet bool
		expectedValue bool
	}{
		{input: "True", expectError: false, expectedIsSet: true, expectedValue: true},
		{input: "true", expectError: false, expectedIsSet: true, expectedValue: true},
		{input: "False", expectError: false, expectedIsSet: true, expectedValue: false},
		{input: "false", expectError: false, expectedIsSet: true, expectedValue: false},
		{input: "null", expectError: false, expectedIsSet: false},
		{input: "nUlL", expectError: false, expectedIsSet: false},
		{input: "", expectError: true},
		{input: "asdfas", expectError: true},
	}

	assert := assert.New(t)

	for _, c := range cases {
		r := root{}
		err := xml.Unmarshal([]byte(fmt.Sprintf(template, c.input)), &r)
		if c.expectError {
			assert.NotNil(err, fmt.Sprintf("Expected error for input '%s', got nil", c.input))
		} else {
			assert.Nil(err)
			assert.Equal(r.Bool.IsSet(), c.expectedIsSet,
				fmt.Sprintf("Expected tag's 'is set' to be %t, got %t", c.expectedIsSet, r.Bool.IsSet()))
			if c.expectedIsSet {
				assert.Equal(r.Bool.Value(), c.expectedValue,
					fmt.Sprintf("Expected tag to have value %t, got %t", c.expectedValue, r.Bool.Value()))
			}
		}
	}
}
