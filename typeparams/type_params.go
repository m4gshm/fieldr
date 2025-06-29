package typeparams

import (
	"errors"
	"fmt"
	"go/types"

	"github.com/m4gshm/fieldr/model/util"
	"github.com/m4gshm/gollections/c"
	"github.com/m4gshm/gollections/k"
	"github.com/m4gshm/gollections/op/delay/string_/join"
	"github.com/m4gshm/gollections/op/string_"
	"github.com/m4gshm/gollections/seq"
	"github.com/m4gshm/gollections/seq2"
)

var ErrNilTypeParam = errors.New("nil type parameter")

type TypeParams seq.Seq[*types.TypeParam]
type NamesConstraints = seq.Seq2[c.KV[string, string], error]

func New(tparams *types.TypeParamList) TypeParams {
	return seq.OfIndexed(tparams.Len(), tparams.At)
}

func (params TypeParams) NamesConstraints(basePkgPath string) NamesConstraints {
	return seq.ToSeq2(params, func(elem *types.TypeParam) (c.KV[string, string], error) {
		if elem == nil {
			return k.V("", ""), ErrNilTypeParam
		}
		return k.V(util.TypeString(elem, basePkgPath), util.TypeString(elem.Constraint(), basePkgPath)), nil
	})
}

func (params TypeParams) Names(basePkgPath string) seq.Seq[string] {
	return seq2.ToSeq(params.NamesConstraints(basePkgPath), func(nameConstraint c.KV[string, string], err error) string {
		if err != nil {
			return fmt.Sprintf("error: %s", err.Error())
		}
		return nameConstraint.K
	})
}

func (params TypeParams) IdentString(basePkgPath string) string {
	return string_.WrapNonEmpty("[", seq.Reduce(params.Names(basePkgPath), join.NonEmpty(", ")), "]")
}

func (params TypeParams) DeclarationString(basePkgPath string) string {
	nameConstraints := seq2.Convert(params.NamesConstraints(basePkgPath), func(nameConstraint c.KV[string, string], err error) (string, string) {
		if err != nil {
			return fmt.Sprintf("/*error: %s*/", err.Error()), ""
		}
		return nameConstraint.K, nameConstraint.V
	})
	joinedStr := seq2.Reduce(nameConstraints, func(prev *c.KV[string, string], name, constraint string) c.KV[string, string] {
		if prev != nil {
			prevConstraint := prev.V
			prevName := prev.K
			delim := ", "
			if constraint != prevConstraint {
				delim = " " + prevConstraint + ", "
			}
			name = prevName + delim + name
		}
		return k.V(name, constraint)
	})
	return string_.WrapNonEmpty("[", string_.JoinNonEmpty(joinedStr.K, " ", joinedStr.V), "]")
}
