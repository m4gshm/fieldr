package params

import (
	"flag"

	"github.com/m4gshm/flag/flagenum"
)

func MultiVal(flagSet *flag.FlagSet, name string, defValues []string, usage string) *[]string {
	return MultiValFixed(flagSet, name, defValues, nil, usage)
}

func MultiValFixed(flagSet *flag.FlagSet, name string, defaulValues, expected []string, usage string) *[]string {
	return flagenum.Wrap(flagSet).MultipleStrings(name, defaulValues, expected, usage)
}
