package stringify_enum

//go:generate fieldr -type EnumWithDuplicates stringify-enum

type EnumWithDuplicates int

const (
	A EnumWithDuplicates = iota
	B
	C
)

func do() {
	const (
		D = C
		E = F
	)
}

const (
	F EnumWithDuplicates = B + C
)
