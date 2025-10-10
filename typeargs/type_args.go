package typeargs

import (
	"go/types"

	"github.com/m4gshm/fieldr/model/util"
	"github.com/m4gshm/gollections/op/delay/string_/join"
	"github.com/m4gshm/gollections/op/string_"
	"github.com/m4gshm/gollections/seq"
)

type TypeArgs struct {
	seq.Seq[types.Type]
	basePkgPath string
}

func New(tparams *types.TypeList, basePkgPath string) TypeArgs {
	return TypeArgs{Seq: seq.OfIndexed(tparams.Len(), tparams.At), basePkgPath: basePkgPath}
}

func (params TypeArgs) Names() seq.Seq[string] {
	return seq.Convert(params.Seq, func(elem types.Type) string {
		if elem == nil {
			return "error: nil type parameter"
		}
		nam := util.TypeString(elem, params.basePkgPath)
		return nam
	})
}

func (params TypeArgs) IdentString() string {
	return string_.WrapNonEmpty("[", params.Names().Reduce(join.NonEmpty(", ")), "]")
}
