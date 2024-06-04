package stringify_enum

//go:generate fieldr -type EnumWithDuplicates stringify-enum
//go:generate fieldr -type Enum stringify-enum

type EnumWithDuplicates int
type Enum int

const (
	A EnumWithDuplicates = iota + 1
	B
	C
	D
)

const (
	AA Enum = iota + 1
	BB
	CC
	DC
)


func F() {
	const (
		E EnumWithDuplicates = iota + D
		F
	)
	_ = F.String()
}

const (
	G EnumWithDuplicates = D + C
)

func FromString(name string) *EnumWithDuplicates {
	e := new(EnumWithDuplicates)
	switch name {
	case "A":
		*e = A
	case "B":
		*e = B
	case "G":
		*e = G
	default:
		e = nil
	}
	return e
}
