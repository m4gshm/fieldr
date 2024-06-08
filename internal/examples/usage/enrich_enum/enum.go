package enrich_enum

//go:generate fieldr -type Enum enrich-enum

type Enum int

const (
	AA Enum = iota + 1
	BB
	CC
	DD
)
