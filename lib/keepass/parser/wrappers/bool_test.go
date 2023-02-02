package wrappers

import (
	"encoding/xml"
	"fmt"
	"testing"
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
		{input: "False", expectError: false, expectedIsSet: true, expectedValue: false},
		{input: "", expectError: true},
		{input: "asdfas", expectError: true},
	}

	for _, c := range cases {
		r := root{}
		err := xml.Unmarshal([]byte(fmt.Sprintf(template, c.input)), &r)
		if c.expectError {
			if err == nil {
				t.Errorf("Expected error for input '%s', got nil", c.input)
			}
		} else {
			if !c.expectError && err != nil {
				t.Errorf("Expected input '%s' to be valid, but got error: %s", c.input, err.Error())
			}
			if r.Bool.IsSet() != c.expectedIsSet {
				t.Errorf("Expected tag's 'is set' to be %t, got %t", c.expectedIsSet, r.Bool.IsSet())
			}
			if r.Bool.Value() != c.expectedValue {
				t.Errorf("Expected tag to have value %t, got %t", c.expectedValue, r.Bool.Value())
			}
		}
	}
}
