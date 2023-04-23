package parser

import (
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/Zaphoood/tresor/src/keepass/parser/wrappers"
)

type Item interface {
	GetUUID() string
	// Performs a 'shallow copy', only copying metadata such as name, UUID, etc. but not subgroups, entries, history etc.
	CopyMeta() Item
}

func (g Group) GetUUID() string {
	return g.UUID
}

func (g Group) CopyMeta() Item {
	gCopy := g
	gCopy.Entries = nil
	gCopy.Groups = nil
	return gCopy
}

func (g *Group) Get(uuid string) (Item, error) {
	for _, group := range g.Groups {
		if group.UUID == uuid {
			return group, nil
		}
	}
	for _, entry := range g.Entries {
		if entry.UUID == uuid {
			return entry, nil
		}
	}
	return nil, fmt.Errorf("Group '%s' has no item with UUID '%s'", g.Name, uuid)
}

func (e Entry) GetUUID() string {
	return e.UUID
}

func (e Entry) CopyMeta() Item {
	e_ := e
	e_.History = nil
	return e_
}

func (e *Entry) Get(key string) (wrappers.Value, error) {
	for _, field := range e.Strings {
		if field.Key == key {
			return field.Value, nil
		}
	}
	return wrappers.Value{}, fmt.Errorf("No such key: %s", key)
}

// TryGet returns the value for the given key if it exists, fallback otherwise
func (e *Entry) TryGet(key, fallback string) string {
	result, err := e.Get(key)
	if err != nil {
		return fallback
	}
	return result.Inner
}

type PathNotFound error

// GetItem returns a group or an item specified by a path of UUIDs. The document is traversed,
// at each level choosing the group with UUID at the current index, until the end of the path is reached.
// The last UUID may be that of an item.
// For an empty path the function will return the top-level groups (which is just one group for most KeePass files)
func (d *Document) GetItem(path []string) (Item, error) {
	current := Group{Groups: d.Root.Groups}

	for i := 0; i < len(path); i++ {
		next, err := current.Get(path[i])
		if err != nil {
			return nil, PathNotFound(fmt.Errorf("Invalid path entry at position %d: %s", i, err))
		}
		switch next := next.(type) {
		case Group:
			current = next
		case Entry:
			if i == len(path)-1 {
				return next, nil
			}
			return nil, errors.New("Got Entry for non-final step in path")
		default:
			return nil, errors.New("Expected Group or Entry from Group.Get()")
		}
	}
	return current, nil
}

// FindPath returns the path to a group with the given UUID if it exists,
// and a bool indicating wether the UUID was found.
func (d *Document) FindPath(uuid string) ([]string, bool) {
	return findPathInGroups(uuid, d.Root.Groups)
}

func findPathInGroups(uuid string, groups []Group) ([]string, bool) {
	for _, group := range groups {
		if group.UUID == uuid {
			return []string{group.UUID}, true
		}
		subpath, found := findPathInGroups(uuid, group.Groups)
		if found {
			return append([]string{group.UUID}, subpath...), true
		}
	}
	return nil, false
}

func (d *Document) GetBinary(id int) ([]byte, error) {
	for _, binary := range d.Meta.Binaries {
		if binary.ID == id {
			decoded, err := base64.StdEncoding.DecodeString(binary.Chardata)
			if err != nil {
				return []byte{}, err
			}
			return decoded, nil
		}
	}
	return []byte{}, fmt.Errorf("No binary with ID: %d", id)
}
