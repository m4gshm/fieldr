package enrich_enum

//go:generate fieldr -type EnumWithDuplicates enrich-enum

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
	F EnumWithDuplicates = B
)
