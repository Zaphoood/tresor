package undo

import (
	"fmt"
	"github.com/Zaphoood/tresor/src/keepass/parser"
)

type UpdateEntryAction struct {
	newEntry parser.Entry
	oldEntry parser.Entry
	// A static value that will be returned on every Do and Undo call
	afterUpdateReturn interface{}
}

func (a UpdateEntryAction) Do(p *parser.Document) interface{} {
	p.UpdateEntry(a.newEntry)
	return a.afterUpdateReturn
}

func (a UpdateEntryAction) Undo(p *parser.Document) interface{} {
	p.UpdateEntry(a.oldEntry)
	return a.afterUpdateReturn
}

func NewUpdateEntryAction(newEntry, oldEntry parser.Entry, returnValue interface{}) UpdateEntryAction {
	if newEntry.UUID != oldEntry.UUID {
		panic(fmt.Sprintf("ERROR: Different UUIDs for old and new entry: '%s' != '%s'", newEntry.UUID, oldEntry.UUID))
	}
	return UpdateEntryAction{newEntry, oldEntry, returnValue}
}
