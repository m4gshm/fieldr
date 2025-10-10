package typeparams

import (
	"errors"
	"fmt"
	"go/types"

	"github.com/m4gshm/gollections/c"
	"github.com/m4gshm/gollections/k"
	"github.com/m4gshm/gollections/op/delay/string_/join"
	"github.com/m4gshm/gollections/op/string_"
	"github.com/m4gshm/gollections/seq"
	"github.com/m4gshm/gollections/seq2"
	"github.com/m4gshm/gollections/slice"

	"github.com/m4gshm/fieldr/model/util"
)

var ErrNilTypeParam = errors.New("nil type parameter")

type TypeParams struct {
	seq.Seq[*types.TypeParam]
	Repack
	basePkgPath string
}
type NameType = c.KV[string, string]
type NameTypes = []NameType
type Repack = func(typ types.Type, basePackagePath string) (types.Type, error)

func New(tparams *types.TypeParamList, repack Repack, basePkgPath string) TypeParams {
	return TypeParams{Seq: seq.OfIndexed(tparams.Len(), tparams.At), Repack: repack, basePkgPath: basePkgPath}
}

func (params TypeParams) nameTypePairs() NameTypes {
	return seq2.ToSeq(seq.Conv(params.Seq, params.elemToNameTypeConv()), func(pair NameType, err error) NameType {
		if err != nil {
			return k.V(fmt.Sprintf("/*error: %s*/", err.Error()), "")
		}
		return pair
	}).Slice()
}

func (params TypeParams) elemToNameTypeConv() func(elem *types.TypeParam) (NameType, error) {
	basePkgPath := params.basePkgPath
	return func(elem *types.TypeParam) (NameType, error) {
		if elem == nil {
			return k.V("", ""), ErrNilTypeParam
		}
		nam := util.TypeString(elem, basePkgPath)
		c, err := params.Repack(elem.Constraint(), basePkgPath)
		if err != nil {
			return k.V("", ""), err
		}
		typ := util.TypeString(c, basePkgPath)
		return k.V(nam, typ), nil
	}
}

func (params TypeParams) IdentDeclNamess() (string, string, []string) {
	pairs := params.nameTypePairs()
	names := names(pairs)
	return identString(names), declarationString(pairs), names
}

func (params TypeParams) Ident() string {
	return identString(names(params.nameTypePairs()))
}

func names(pairs NameTypes) []string {
	return slice.Convert(pairs, c.KV[string, string].Key)
}

func identString(names []string) string {
	return string_.WrapNonEmpty("[", slice.Reduce(names, join.NonEmpty(", ")), "]")
}

func declarationString(pairs NameTypes) string {
	joinedStr := slice.Reduce(pairs, func(prev NameType, pair NameType) NameType {
		prevType := prev.V
		prevName := prev.K
		delim := ", "
		name := pair.K
		typ := pair.V
		if typ != prevType {
			delim = " " + prevType + ", "
		}
		name = prevName + delim + name
		return k.V(name, typ)
	})
	return string_.WrapNonEmpty("[", string_.JoinNonEmpty(joinedStr.K, " ", joinedStr.V), "]")
}
