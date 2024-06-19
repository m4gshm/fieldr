package generator

import (
	"path"
	"unicode"
	"unicode/utf8"

	"github.com/m4gshm/gollections/predicate/is"
	"github.com/m4gshm/gollections/slice"
)

func badSymbol(ch rune) bool {
	return !('a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' ||
		ch == '_' || ch >= utf8.RuneSelf && (unicode.IsLetter(ch)))
}

func packagePathToName(importPath string) string {
	base := path.Base(importPath)
	pathName := string(slice.Filter([]rune(base), is.Not(badSymbol)))
	return pathName
}
