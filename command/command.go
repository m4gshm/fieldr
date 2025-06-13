package command

import (
	"flag"
	"fmt"
	"os"

	"github.com/m4gshm/gollections/convert/as"
	"github.com/m4gshm/gollections/expr/get"
	"github.com/m4gshm/gollections/op"
	"github.com/m4gshm/gollections/slice"
)

func New(name, description string, flagSet *flag.FlagSet, op func(context *Context) error) *Command {
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
	op                        func(context *Context) error
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

func (c *Command) Run(context *Context) error {
	return c.op(context)
}

func (c *Command) Parse(arguments []string) ([]string, error) {
	if err := c.flagSet.Parse(arguments); err != nil {
		return nil, fmt.Errorf("parse args '%s': %w", c.name, err)
	}
	return c.flagSet.Args(), nil
}

func Get(name string) *Command {
	c := index[name]
	return get.If(c != nil, c).Else(nil)
}

func Supported() []string {
	return slice.Convert(commands, getCommandFuncName)
}

func PrintUsage() {
	out := os.Stderr
	_, _ = fmt.Fprintln(out, "Commands:")

	for _, cmd := range slice.Convert(commands, op.Get[*Command]) {
		_, _ = fmt.Fprintln(out, "  "+cmd.name+"\n    \t"+cmd.description)
	}
}

var commands = []func() *Command{
	NewFieldsToConsts,
	NewAsMapMethod,
	NewNewOpt,
	NewNewFull,
	NewBuilderStruct,
	NewGettersSetters,
	NewEnrichConstType,
}

var index = slice.Map(commands, getCommandFuncName, as.Is)

func getCommandFuncName(c func() *Command) string { return c().Name() }
