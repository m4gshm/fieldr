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
type NamesConstraints = seq.SeqE[c.KV[string, string]]
type StrKV = c.KV[string, string]

func New(tparams *types.TypeParamList) TypeParams {
	return TypeParams(seq.OfIndexed(tparams.Len(), tparams.At))
}

func (params TypeParams) NamesConstraints(basePkgPath string) NamesConstraints {
	return seq.Conv(params, func(elem *types.TypeParam) (StrKV, error) {
		if elem == nil {
			return k.V("", ""), ErrNilTypeParam
		}
		return k.V(util.TypeString(elem, basePkgPath), util.TypeString(elem.Constraint(), basePkgPath)), nil
	})
}

func (params TypeParams) Names(basePkgPath string) seq.Seq[string] {
	return seq2.ToSeq(params.NamesConstraints(basePkgPath), func(nameConstraint StrKV, err error) string {
		if err != nil {
			return fmt.Sprintf("error: %s", err.Error())
		}
		return nameConstraint.K
	})
}

func (params TypeParams) IdentString(basePkgPath string) string {
	return string_.WrapNonEmpty("[", params.Names(basePkgPath).Reduce(join.NonEmpty(", ")), "]")
}

func (params TypeParams) DeclarationString(basePkgPath string) string {
	nameConstraints := seq2.Convert(params.NamesConstraints(basePkgPath), func(nameConstraint StrKV, err error) (string, string) {
		if err != nil {
			return fmt.Sprintf("/*error: %s*/", err.Error()), ""
		}
		return nameConstraint.K, nameConstraint.V
	})
	joinedStr := seq2.Reduce(nameConstraints, func(prev *StrKV, name, constraint string) StrKV {
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
