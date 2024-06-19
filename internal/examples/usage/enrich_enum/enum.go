package enrich_enum

//go:generate fieldr -type Enum enrich-const-type -export

type Enum int

const (
	AA Enum = iota + 1
	BB
	CC
	DD
)

//go:generate fieldr -type StringEnum enrich-const-type -export

type BaseStringEnum string
type StringEnum BaseStringEnum

const (
	FIRST  StringEnum = "first one"
	SECOND StringEnum = "one more"
	THIRD  StringEnum = "any third"
)
