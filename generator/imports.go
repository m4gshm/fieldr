package generator

import (
	"unicode"
	"unicode/utf8"

	"github.com/m4gshm/gollections/predicate/is"
	"github.com/m4gshm/gollections/slice"

	"github.com/m4gshm/fieldr/model/util"
)

func badSymbol(ch rune) bool {
	return !('a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' ||
		ch == '_' || ch >= utf8.RuneSelf && (unicode.IsLetter(ch)))
}

func packagePathToName(importPath string) string {
	base := util.GetPackageName(importPath)
	pathName := string(slice.Filter([]rune(base), is.Not(badSymbol)))
	return pathName
}
