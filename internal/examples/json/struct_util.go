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
		structJsonAddress}
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

func (s *Struct[S]) val(f structJson) any {
	if s == nil {
		return nil
	}
	switch f {
	case structJsonID:
		if ms := s.MiddleStruct; ms != nil {
			if ida := ms.BaseStruct.IDAware; ida != nil {
				return ida.ID
			}
		}
	case structJsonName:
		return s.Name
	case structJsonSurname:
		return s.Surname
	case structJsonNoTag:
		return s.NoTag
	case structJsonAddress:
		if a := s.Address; a != nil {
			return a
		}
	}
	return nil
}
