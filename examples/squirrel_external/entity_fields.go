package squirrel_external

import "example/squirrel"

type Col string

const (
	colID      Col = "ID"
	colName    Col = "NAME"
	colSurname Col = "SURNAME"
	colVersion Col = "version"
)

func cols() []Col { //nolint
	return []Col{
		colID,
		colName,
		colSurname,
		colVersion,
	}
}

func (c Col) field() string { //nolint
	switch c {
	case colID:
		return "ID"
	case colName:
		return "Name"
	case colSurname:
		return "Surname"
	case colVersion:
		return "Versioned.Version"
	}
	return ""
}

func val(s *squirrel.Entity, f Col) interface{} { //nolint
	if s == nil {
		return nil
	}
	switch f {
	case colID:
		return s.ID
	case colName:
		return s.Name
	case colSurname:
		return s.Surname
	case colVersion:
		return s.Versioned.Version
	}
	return nil
}

func ref(s *squirrel.Entity, f Col) interface{} { //nolint
	if s == nil {
		return nil
	}
	switch f {
	case colID:
		return &s.ID
	case colName:
		return &s.Name
	case colSurname:
		return &s.Surname
	case colVersion:
		return &s.Versioned.Version
	}
	return nil
}
