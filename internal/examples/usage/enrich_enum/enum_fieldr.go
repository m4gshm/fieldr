// Code generated by 'fieldr'; DO NOT EDIT.

package enrich_enum

func (e Enum) String() string {
	switch e {
	case 1:
		return "AA"
	case 2:
		return "BB"
	case 3:
		return "CC"
	case 4:
		return "DD"
	default:
		return ""
	}
}

func EnumValues() []Enum {
	return []Enum{
		AA,
		BB,
		CC,
		DD,
	}
}

func EnumFromString(s string) (e Enum, ok bool) {
	ok = true
	switch s {
	case "AA":
		e = AA
	case "BB":
		e = BB
	case "CC":
		e = CC
	case "DD":
		e = DD
	default:
		ok = false
	}
	return
}