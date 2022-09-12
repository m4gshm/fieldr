package squirrel_external

import (
	"example/squirrel"
)

type (
	Col string
)

const (
	colID               = Col("ID")
	colName             = Col("NAME")
	colSurname          = Col("SURNAME")
	colVersionedVersion = Col("version")
)

func cols() []Col {
	return []Col{
		colID,
		colName,
		colSurname,
		colVersionedVersion,
	}
}
func (c Col) field() string {
	switch c {
	case colID:
		return "ID"
	case colName:
		return "Name"
	case colSurname:
		return "Surname"
	case colVersionedVersion:
		return "Versioned.Version"
	}
	return ""
}

func (c Col) val(s *squirrel.Entity) interface{} {
	switch c {
	case colID:
		return s.ID
	case colName:
		return s.Name
	case colSurname:
		return s.Surname
	case colVersionedVersion:
		return s.Versioned.Version
	}
	return nil
}

func (c Col) ref(s *squirrel.Entity) interface{} {
	switch c {
	case colID:
		return &s.ID
	case colName:
		return &s.Name
	case colSurname:
		return &s.Surname
	case colVersionedVersion:
		return &s.Versioned.Version
	}
	return nil
}
