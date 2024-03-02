package utils

import "fmt"

type Test[T any, K any] struct {
	Name  string
	Input T
	Want  K
}

func (t Test[T, K]) ToString(result K) string {
	return fmt.Sprintf("%s. Have: %v, want: %v", t.Name, result, t.Want)
}
