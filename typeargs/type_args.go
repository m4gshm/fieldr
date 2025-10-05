package typeargs

import (
	"go/types"

	"github.com/m4gshm/fieldr/model/util"
	"github.com/m4gshm/gollections/op/delay/string_/join"
	"github.com/m4gshm/gollections/op/string_"
	"github.com/m4gshm/gollections/seq"
)


type TypeArgs seq.Seq[types.Type]

func New(tparams *types.TypeList) TypeArgs {
	return TypeArgs(seq.OfIndexed(tparams.Len(), tparams.At))
}

func (params TypeArgs) Names(basePkgPath string) seq.Seq[string] {
	return seq.Convert(params, func(elem types.Type) string {
		if elem == nil {
			return "error: nil type parameter"
		}
		nam := util.TypeString(elem, basePkgPath)
		return nam
	})
}

func (params TypeArgs) IdentString(basePkgPath string) string {
	return string_.WrapNonEmpty("[", params.Names(basePkgPath).Reduce(join.NonEmpty(", ")), "]")
}
