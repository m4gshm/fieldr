package generator

import (
	"strconv"

	"github.com/m4gshm/gollections/collection/mutable"
)

func NewUniqueShortVarGenerator(shortVar string) *UniqueShortVarGenerator {
	u := &UniqueShortVarGenerator{}
	u.Get(shortVar)
	return u
}

type UniqueShortVarGenerator struct {
	uniqueVars *mutable.Set[string]
}

func (u *UniqueShortVarGenerator) Get(shortVar string) string {
	if u.uniqueVars == nil {
		u.uniqueVars = mutable.NewSet[string]()
	}
	for i := 1; !u.uniqueVars.AddNew(shortVar); i++ {
		shortVar += strconv.Itoa(i)
	}
	return shortVar
}
