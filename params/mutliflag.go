package params

import (
	"flag"
	"fmt"
	"strings"
)

var void struct{}

type multiflag struct {
	name           string
	values         []string
	expected       []string
	uniques        map[string]struct{}
	unusedDefaults map[string]struct{}
	expectedUniq   map[string]struct{}
}

var _ flag.Value = (*multiflag)(nil)

func (f *multiflag) String() string {
	return strings.Join(f.values, ",")
}

func (f *multiflag) Set(s string) error {
	if _, ok := f.unusedDefaults[s]; ok {
		delete(f.unusedDefaults, s)
		return nil
	}
	if err := checkDuplicated("", s, f.uniques, f.name); err != nil {
		return err
	} else if len(f.expectedUniq) > 0 {
		if _, ok := f.expectedUniq[s]; !ok {
			return fmt.Errorf("flag %s: invalid value '%s', expected %s", f.name, s, strings.Join(f.expected, ", "))
		}
	}

	f.values = append(f.values, s)
	f.uniques[s] = void
	return nil
}

func (f *multiflag) Get() interface{} {
	return f.values
}

func MultiVal(flagSet *flag.FlagSet, name string, defValues []string, usage string) *[]string {
	return MultiValFixed(flagSet, name, defValues, nil, usage)
}

func MultiValFixed(flagSet *flag.FlagSet, name string, defaulValues, expected []string, usage string) *[]string {
	expecteUniq := map[string]struct{}{}
	for _, e := range expected {
		if err := checkDuplicated("expected", e, expecteUniq, name); err != nil {
			panic(err)
		}
		expecteUniq[e] = void
	}
	uniques := map[string]struct{}{}
	unusedDefaults := map[string]struct{}{}
	for _, defValue := range defaulValues {
		if err := checkDuplicated("default", defValue, uniques, name); err != nil {
			panic(err)
		}
		uniques[defValue] = void
		unusedDefaults[defValue] = void
	}
	values := multiflag{
		name: name, values: defaulValues, expected: expected, uniques: uniques,
		unusedDefaults: unusedDefaults, expectedUniq: expecteUniq,
	}
	suffix := ""
	if len(expecteUniq) > 0 {
		suffix = "(expected: " + strings.Join(expected, ", ") + ")"
	}
	if len(usage) > 0 {
		suffix = " " + suffix
	}
	flagSet.Var(&values, name, usage+suffix)
	return &values.values
}

func checkDuplicated(typeVal, value string, duplicateControl map[string]struct{}, name string) error {
	if _, ok := duplicateControl[value]; !ok {
		return nil
	}
	if len(typeVal) > 0 {
		typeVal += " "
	}
	return fmt.Errorf("duplicated %svalue '%s' of parameter '%s'", typeVal, value, name)
}
