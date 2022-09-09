package coderewriter

import "github.com/m4gshm/fieldr/generator"

func New(fieldValueRewriters []string) (*generator.CodeRewriter, error) {
	return generator.NewCodeRewriter(fieldValueRewriters)
}
