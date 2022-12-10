package use

import (
	"fmt"
	"go/ast"
	"go/token"
)

func Err(message string) *Error {
	return &Error{message: message}
}

// func CmdErr(message string) *Error {
// 	return &Error{message: message}
// }

func FileCommentErr(message string, astFile *ast.File, tokenFile *token.File, comment *ast.Comment) *Error {
	return &Error{message: message, comment: comment, astFile: astFile, tokenFile: tokenFile}
}

type Error struct {
	message string

	astFile   *ast.File
	tokenFile *token.File
	comment   *ast.Comment
}

func (e *Error) Error() string {
	m := e.message
	if e.tokenFile != nil {
		n := e.tokenFile.Name()
		m += fmt.Sprintf(" file: %s", n)

		if e.comment != nil {
			p := e.tokenFile.Position(e.comment.Pos())
			m += fmt.Sprintf(" line %d, col %d", p.Line, p.Column)
		}
	}
	return m
}
