package undo

import (
	"fmt"
	"github.com/Zaphoood/tresor/src/keepass/parser"
)

type UpdateEntryAction struct {
	newEntry parser.Entry
	oldEntry parser.Entry
}

func (a UpdateEntryAction) Do(p *parser.Document) {
	p.UpdateEntry(a.newEntry)
}

func (a UpdateEntryAction) Undo(p *parser.Document) {
	p.UpdateEntry(a.oldEntry)
}

func NewUpdateEntryAction(newEntry, oldEntry parser.Entry) UpdateEntryAction {
	if newEntry.UUID != oldEntry.UUID {
		panic(fmt.Sprintf("ERROR: Different UUIDs for old and new entry: '%s' != '%s'", newEntry.UUID, oldEntry.UUID))
	}
	return UpdateEntryAction{newEntry, oldEntry}
}
