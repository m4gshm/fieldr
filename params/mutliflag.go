package params

import (
	"flag"
	"fmt"
	"strings"
)

var void struct{}

type multiflag struct {
	name                  string
	values, expected      []string
	uniques, expectedUniq map[string]struct{}
}

var _ flag.Value = (*multiflag)(nil)

func (f *multiflag) String() string {
	return strings.Join(f.values, ",")
}

func (f *multiflag) Set(s string) error {
	if f.values == nil {
		f.values = []string{}
	}

	if err := checkDuplicated(s, f.uniques, f.name); err != nil {
		return err
	}

	if len(f.expectedUniq) > 0 {
		if _, ok := f.expectedUniq[s]; !ok {
			return fmt.Errorf("flag %s: invalid value '%s', expected %s", f.name, s, strings.Join(f.expected, ", "))
		}
	}

	f.values = append(f.values, s)
	f.uniques[s] = void
	return nil
}

func (f *multiflag) Get() interface{} { return f.values }

func MultiVal(flagSet *flag.FlagSet, name string, defValues []string, usage string) *[]string {
	return MultiValFixed(flagSet, name, defValues, nil, usage)
}

func MultiValFixed(flagSet *flag.FlagSet, name string, defValues, expected []string, usage string) *[]string {
	expecteUniq := map[string]struct{}{}
	for _, e := range expected {
		if err := checkDuplicated(e, expecteUniq, name); err != nil {
			panic(err)
		}
		expecteUniq[e] = void
	}
	uniques := map[string]struct{}{}
	for _, defValue := range defValues {
		if err := checkDuplicated(defValue, uniques, name); err != nil {
			panic(err)
		}
		uniques[defValue] = void
	}
	values := multiflag{name: name, values: defValues, expected: expected, uniques: uniques, expectedUniq: expecteUniq}
	suffix := ""
	if len(expecteUniq) > 0 {
		suffix = "(expected: " + strings.Join(expected, ", ") + ")"
	}
	flagSet.Var(&values, name, usage+suffix)
	return &values.values
}

func checkDuplicated(value string, duplicateControl map[string]struct{}, name string) error {
	_, ok := duplicateControl[value]
	if ok {
		return fmt.Errorf("duplicated value %v of parameter %v ", value, name)
	}
	return nil
}
