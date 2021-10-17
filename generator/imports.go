package generator

import (
	"path"
	"strings"
	"unicode"
	"unicode/utf8"
)

func badSymbol(ch rune) bool {
	return !('a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' ||
		ch == '_' || ch >= utf8.RuneSelf && (unicode.IsLetter(ch)))
}

func pkgPathToName(importPath string) string {
	base := path.Base(importPath)
	builder := strings.Builder{}
	for _, r := range base {
		if badSymbol(r) {
			builder.WriteString("_")
		} else {
			builder.WriteRune(r)
		}
	}
	pathName := builder.String()
	return pathName
}
