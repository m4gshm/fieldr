package command

import (
	"flag"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/struc"
)

type Command struct {
	Usage string
	Op    func(g *generator.Generator, m *struc.Model) error
	Flag  *flag.FlagSet
}

var commands = map[string]func() *Command{
	"enum-const": NewEnumConst,
}

func Get(name string) func() *Command {
	return commands[name]
}
