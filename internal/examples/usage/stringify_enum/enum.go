package stringify_enum

//go:generate fieldr -type Enum stringify-enum -export

type Enum int

const (
	AA Enum = iota + 1
	BB
	CC
	DD
)
