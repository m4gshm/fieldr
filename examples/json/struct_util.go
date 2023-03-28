package json

type structJson string

const (
	structJsonID      structJson = "id"
	structJsonName    structJson = "name,omitempty"
	structJsonSurname structJson = "surname,omitempty"
	structJsonNoTag   structJson = "NoTag"
	structJsonAddress structJson = "address,omitempty"
)

func structJsons() []structJson {
	return []structJson{
		structJsonID,
		structJsonName,
		structJsonSurname,
		structJsonNoTag,
		structJsonAddress,
	}
}

func (c structJson) field() string {
	switch c {
	case structJsonID:
		return "MiddleStruct.BaseStruct.IDAware.ID"
	case structJsonName:
		return "Name"
	case structJsonSurname:
		return "Surname"
	case structJsonNoTag:
		return "NoTag"
	case structJsonAddress:
		return "Address"
	}
	return ""
}

func (s *Struct[S]) val(f structJson) interface{} {
	if s == nil {
		return nil
	}
	switch f {
	case structJsonID:
		if s_ms := s.MiddleStruct; s_ms != nil {
			if s__bs_ida := s_ms.BaseStruct.IDAware; s__bs_ida != nil {
				return s__bs_ida.ID
			}
		}
	case structJsonName:
		return s.Name
	case structJsonSurname:
		return s.Surname
	case structJsonNoTag:
		return s.NoTag
	case structJsonAddress:
		if s_a := s.Address; s_a != nil {
			return s_a
		}
	}
	return nil
}
