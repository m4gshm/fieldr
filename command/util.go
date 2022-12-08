package command

import (
	"github.com/m4gshm/gollections/immutable"
	"github.com/m4gshm/gollections/immutable/set"
)

func toSet(values []string) immutable.Set[string] {
	return set.New[string](values)
}
