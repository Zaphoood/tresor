package undo

import "github.com/Zaphoood/tresor/src/util"

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
	managed *T
	actions []Action[T]
	// step is an index into actions which points at the action after last executed action
	step int
}

func NewUndoManager[T any](initial *T) UndoManager[T] {
	return UndoManager[T]{
		managed: initial,
		actions: []Action[T]{},
		step:    0,
	}
}

func (u *UndoManager[T]) Get() T {
	return *u.managed
}

func (u *UndoManager[T]) Set(value T) {
	*u.managed = value
}

func (u *UndoManager[T]) Do(action Action[T]) {
	action.Do(u.managed)
	u.actions = append(u.actions[:util.Max(u.step, len(u.actions))], action)
	u.step++
}

func (u *UndoManager[T]) Undo() error {
	if u.step == 0 {
		return AtLastChange{}
	}
	u.step--
	u.actions[u.step].Undo(u.managed)
	return nil
}

func (u *UndoManager[T]) Redo() error {
	if u.step >= len(u.actions) {
		return AtNewestChange{}
	}
	u.actions[u.step].Do(u.managed)
	u.step++
	return nil
}
