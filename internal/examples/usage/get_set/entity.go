package get_set

import "time"

//go:generate fieldr -type Entity get-set

type BaseEntity[ID any] struct {
	id ID
}

type Entity[ID any] struct {
	*BaseEntity[ID]
	name    string
	surname string
	ts      time.Time
}
