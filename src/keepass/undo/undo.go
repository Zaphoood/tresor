package undo

import (
	"github.com/Zaphoood/tresor/src/util"
)

type Action[T any] interface {
	Do(*T)
	Undo(*T)
}

type AtLastChange struct{}

func (_ AtLastChange) Error() string {
	return "Already at last change"
}

type AtNewestChange struct{}

func (_ AtNewestChange) Error() string {
	return "Already at newest change"
}

type UndoManager[T any] struct {
	actions []Action[T]
	// step is an index into actions which points at the action after last executed action
	step int
}

func NewUndoManager[T any]() UndoManager[T] {
	return UndoManager[T]{
		actions: []Action[T]{},
		step:    0,
	}
}
func (u *UndoManager[T]) Do(target *T, action Action[T]) {
	action.Do(target)
	u.actions = append(u.actions[:util.Min(u.step, len(u.actions))], action)
	u.step++
}

func (u *UndoManager[T]) Undo(target *T) error {
	if u.step == 0 {
		return AtLastChange{}
	}
	u.step--
	u.actions[u.step].Undo(target)
	return nil
}

func (u *UndoManager[T]) Redo(target *T) error {
	if u.step >= len(u.actions) {
		return AtNewestChange{}
	}
	u.actions[u.step].Do(target)
	u.step++
	return nil
}
