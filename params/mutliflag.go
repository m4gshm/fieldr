package params

import (
	"flag"
	"fmt"
	"strings"
)

var void interface{}

type multiflag struct {
	name             string
	values           []string
	duplicateControl map[string]interface{}
}

func (f *multiflag) String() string {
	return strings.Join(f.values, ",")
}

func (f *multiflag) Set(s string) error {
	if f.values == nil {
		f.values = []string{}
	}

	if err := checkDuplicated(s, f.duplicateControl, f.name); err != nil {
		return err
	}
	f.values = append(f.values, s)
	f.duplicateControl[s] = void
	return nil
}

func (f *multiflag) Get() interface{} { return f.values }

func multiVal(flagSet *flag.FlagSet, name string, defValues []string, usage string) *[]string {
	duplicateControl := map[string]interface{}{}
	for _, defValue := range defValues {
		if err := checkDuplicated(defValue, duplicateControl, name); err != nil {
			panic(err)
		}
		duplicateControl[defValue] = void
	}
	values := multiflag{name: name, values: defValues, duplicateControl: duplicateControl}
	flagSet.Var(&values, name, usage)
	return &values.values
}

func checkDuplicated(value string, duplicateControl map[string]interface{}, name string) error {
	_, ok := duplicateControl[value]
	if ok {
		return fmt.Errorf("Duplicated value %v of parameter %v ", value, name)
	}
	return nil
}
