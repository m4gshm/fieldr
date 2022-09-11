package command

import (
	"flag"
	"fmt"
	"os"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/struc"
)

func New(name, description string, flagSet *flag.FlagSet, op func(g *generator.Generator, m *struc.HierarchicalModel) error) *Command {
	c := &Command{
		name:        name,
		description: description,
		flagSet:     flagSet,
		op:          op,
	}
	flagSet.Usage = c.PrintUsage
	return c
}

type Command struct {
	name, description, manual string
	op                        func(g *generator.Generator, m *struc.HierarchicalModel) error
	flagSet                   *flag.FlagSet
}

func (c *Command) Name() string {
	return c.name
}

func (c *Command) PrintUsage() {
	out := c.flagSet.Output()
	_, _ = fmt.Fprintln(out, "Command "+c.name)
	_, _ = fmt.Fprintln(out, "\t"+c.description)
	_, _ = fmt.Fprintln(out, "Flags:")
	c.flagSet.PrintDefaults()
	if len(c.manual) > 0 {
		_, _ = fmt.Fprintln(out, c.manual)
	}
}

func (c *Command) Run(g *generator.Generator, m *struc.HierarchicalModel) error {
	return c.op(g, m)
}

func (c *Command) Parse(arguments []string) ([]string, error) {
	if err := c.flagSet.Parse(arguments); err != nil {
		return nil, fmt.Errorf("parse args '%s': %w", c.name, err)
	}
	return c.flagSet.Args(), nil
}

func Get(name string) *Command {
	c := index[name]
	if c != nil {
		return c()
	}
	return nil
}

func Supported() []string {
	list := []string{}
	for _, cmd := range commands {
		list = append(list, cmd().name)
	}
	return list
}

func PrintUsage() {
	out := os.Stderr
	_, _ = fmt.Fprintln(out, "Commands:")
	for _, cmd := range commands {
		c := cmd()
		_, _ = fmt.Fprintln(out, "  "+c.name+"\n    \t"+c.description)
	}
}

var commands = []func() *Command{
	NewEnumConst,
	NewAsMapMethod,
}

var index = toMap(commands)

func toMap(commands []func() *Command) map[string]func() *Command {
	index := map[string]func() *Command{}
	for _, c := range commands {
		index[c().name] = c
	}
	return index
}
