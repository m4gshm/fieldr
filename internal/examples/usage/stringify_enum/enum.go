package stringify_enum

//go:generate fieldr -type Enum stringify-enum

type Enum int

const (
	AA Enum = iota + 1
	BB
	CC
	DC
)
