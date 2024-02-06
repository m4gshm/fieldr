// Code generated by 'fieldr'; DO NOT EDIT.

package squirrel

const (
	AlterColID      Col = "ID"
	AlterColName    Col = "NAME"
	AlterColSurname Col = "SURNAME"
)

func ACols() []Col {
	return []Col{AlterColID, AlterColName, AlterColSurname}
}

func (s *Entity) Aval(f Col) any {
	if s == nil {
		return nil
	}
	switch f {
	case AlterColID:
		return s.ID
	case AlterColName:
		return s.Name
	case AlterColSurname:
		return s.Surname
	}
	return nil
}

func (s *Entity) Aref(f Col) any {
	if s == nil {
		return nil
	}
	switch f {
	case AlterColID:
		return &s.ID
	case AlterColName:
		return &s.Name
	case AlterColSurname:
		return &s.Surname
	}
	return nil
}
