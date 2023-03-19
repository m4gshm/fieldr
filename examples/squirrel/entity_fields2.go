// Code generated by 'fieldr'; DO NOT EDIT.

package squirrel

type Col2 string

const (
	col2ID   Col2 = "ID"
	col2Name Col2 = "NAME"
)

func col2s() []Col2 {
	return []Col2{col2ID, col2Name}
}

func (s *Entity2) val(f Col2) interface{} {
	if s == nil {
		return nil
	}
	switch f {
	case col2ID:
		return s.ID
	case col2Name:
		return s.Name
	}
	return nil
}

func (s *Entity2) ref(f Col2) interface{} {
	if s == nil {
		return nil
	}
	switch f {
	case col2ID:
		return &s.ID
	case col2Name:
		return &s.Name
	}
	return nil
}
