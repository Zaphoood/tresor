package util

//import (
//	"golang.org/x/exp/constraints"
//)
//
//func Max[T constraints.Ordered](a T, b T) T {
//	if a > b {
//		return a
//	}
//	return b
//}

// TODO: Use generic version of this function
func Max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
