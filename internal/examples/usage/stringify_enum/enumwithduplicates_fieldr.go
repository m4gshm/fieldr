// Code generated by 'fieldr'; DO NOT EDIT.

package stringify_enum

func (e EnumWithDuplicates) String() []string {
	switch e {
	case 0:
		return []string{"A"}
	case 1:
		return []string{"B", "F"}
	case 2:
		return []string{"C"}
	default:
		return nil
	}
}

func EnumWithDuplicatesValues() []EnumWithDuplicates {
	return []EnumWithDuplicates{
		A,
		B, //F
		C,
	}
}

func EnumWithDuplicatesFromString(s string) (r EnumWithDuplicates, ok bool) {
	ok = true
	switch s {
	case "A":
		r = A
	case "B", "F":
		r = B
	case "C":
		r = C
	default:
		ok = false
	}
	return
}
