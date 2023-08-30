package undo

type Action[T any] interface {
	Do(*T) interface{}
	Undo(*T) interface{}
	Description() string
}

type AtOldestChange struct{}

func (_ AtOldestChange) Error() string {
	return "Already at oldest change"
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
func (u *UndoManager[T]) Do(target *T, action Action[T]) interface{} {
	u.actions = append(u.actions[:u.step], action)
	u.step++
	return action.Do(target)
}

func (u *UndoManager[T]) Undo(target *T) (interface{}, error) {
	if u.step == 0 {
		return nil, AtOldestChange{}
	}
	u.step--
	return u.actions[u.step].Undo(target), nil
}

func (u *UndoManager[T]) Redo(target *T) (interface{}, error) {
	if u.step >= len(u.actions) {
		return nil, AtNewestChange{}
	}
	result := u.actions[u.step].Do(target)
	u.step++
	return result, nil
}
