package use

import (
	"fmt"
	"go/ast"
)

func Err(message string) *Error {
	return &Error{message: message}
}

// func CmdErr(message string) *Error {
// 	return &Error{message: message}
// }

func FileCommentErr(message string, file *ast.File, comment *ast.Comment) *Error {
	return &Error{message: message, comment: comment, file: file}
}

type Error struct {
	message string

	file    *ast.File
	comment *ast.Comment
}

func (e *Error) Error() string {
	m := e.message
	if e.file != nil {
		n := e.file.Name.String()
		m += fmt.Sprintf(" file: %s", n)
	}
	if e.comment != nil {
		m += fmt.Sprintf(" pos: %d", e.comment.Pos())
	}
	return m
}
