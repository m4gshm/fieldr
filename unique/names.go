package unique

import (
	"strconv"

	"github.com/m4gshm/gollections/collection/mutable"
	"github.com/m4gshm/gollections/seq"
)

func NewNamesWith(opts ...func(*Names)) *Names {
	u := &Names{calc: increment(1, 1)}
	for _, o := range opts {
		o(u)
	}
	return u
}

func PreInit(suffix ...string) func(*Names) {
	return func(un *Names) {
		seq.ForEach(seq.Of(suffix...), un.Add)
	}
}

func DistinctBySuffix(suffix string) func(*Names) {
	return func(un *Names) {
		un.calc = addSuffix(suffix)
	}
}

type Names struct {
	uniques *mutable.Set[string]
	calc    func(u *Names, varName string) string
}

func (u *Names) Get(varName string) string {
	if u != nil {
		if u.uniques == nil {
			u.uniques = mutable.NewSet[string]()
		}
		varName = u.calc(u, varName)
	}
	return varName
}

func (u *Names) Add(varName string) {
	u.Get(varName)
}

func increment(first, delta int) func(u *Names, varName string) string {
	return func(u *Names, varName string) string {
		for i := first; !u.uniques.AddNew(varName); i = i + delta {
			varName += strconv.Itoa(i)
		}
		return varName
	}
}

func addSuffix(suffix string) func(u *Names, varName string) string {
	return func(u *Names, varName string) string {
		for i := 1; !u.uniques.AddNew(varName); i++ {
			varName += varName + suffix
		}
		return varName
	}
}