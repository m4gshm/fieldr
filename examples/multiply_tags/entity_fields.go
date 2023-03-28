package multiply_tags

type EntityCol string

const (
	ENTITY_COL_ID          EntityCol = "ID"
	ENTITY_COL_NAME        EntityCol = "NAME"
	ENTITY_COL_SURNAME     EntityCol = "SURNAME"
	ENTITY_COL_UPDATED_AT  EntityCol = "UPDATED_AT"
	ENTITY_COL_UPDATED_AT2 EntityCol = "UPDATED_AT2"
	ENTITY_COL_UPDATED_AT3 EntityCol = "UPDATED_AT3"
	EntityGormID                     = "ID"
	EntityGormName                   = "NAME"
	EntityGormSurname                = "SURNAME"
	EntityGormUpd                    = "UPD"
	EntityGormUpd2                   = "UPD2"
	EntityGormUpd3                   = "UPD3"
	EntityJsonID                     = "id"
	EntityJsonName                   = "name"
	EntityGormJsonID                 = "id"
	EntityGormJsonName               = "NAME"
	EntityGormJsonSurname            = "SURNAME"
)

func entityCols() []EntityCol { //nolint
	return []EntityCol{
		ENTITY_COL_ID,
		ENTITY_COL_NAME,
		ENTITY_COL_SURNAME,
		ENTITY_COL_UPDATED_AT,
		ENTITY_COL_UPDATED_AT2,
		ENTITY_COL_UPDATED_AT3,
	}
}

func (s *Entity) val(f EntityCol) interface{} { //nolint
	if s == nil {
		return nil
	}
	switch f {
	case ENTITY_COL_ID:
		if s_be := s.BaseEntity; s_be != nil {
			return s_be.ID
		}
	case ENTITY_COL_NAME:
		return s.Name
	case ENTITY_COL_SURNAME:
		return s.Surname
	case ENTITY_COL_UPDATED_AT:
		if s_u := s.Upd; s_u != nil {
			if _s_ := *s_u; _s_ != nil {
				if __s_ := *_s_; __s_ != nil {
					if ___s_ := *__s_; ___s_ != nil {
						return ___s_.UpdatedAt
					}
				}
			}
		}
	case ENTITY_COL_UPDATED_AT2:
		if s_u2 := s.Upd2; s_u2 != nil {
			if _s_2 := *s_u2; _s_2 != nil {
				if __s_2 := *_s_2; __s_2 != nil {
					if ___s_2 := *__s_2; ___s_2 != nil {
						return ___s_2.UpdatedAt2
					}
				}
			}
		}
	case ENTITY_COL_UPDATED_AT3:
		if s_u3 := s.Upd3; s_u3 != nil {
			if _s_3 := *s_u3; _s_3 != nil {
				return _s_3.UpdatedAt3
			}
		}
	}
	return nil
}

func (s *Entity) ref(f EntityCol) interface{} { //nolint
	if s == nil {
		return nil
	}
	switch f {
	case ENTITY_COL_ID:
		if s_be := s.BaseEntity; s_be != nil {
			return &s_be.ID
		}
	case ENTITY_COL_NAME:
		return &s.Name
	case ENTITY_COL_SURNAME:
		return &s.Surname
	case ENTITY_COL_UPDATED_AT:
		if s_u := s.Upd; s_u != nil {
			if _s_ := *s_u; _s_ != nil {
				if __s_ := *_s_; __s_ != nil {
					if ___s_ := *__s_; ___s_ != nil {
						return &___s_.UpdatedAt
					}
				}
			}
		}
	case ENTITY_COL_UPDATED_AT2:
		if s_u2 := s.Upd2; s_u2 != nil {
			if _s_2 := *s_u2; _s_2 != nil {
				if __s_2 := *_s_2; __s_2 != nil {
					if ___s_2 := *__s_2; ___s_2 != nil {
						return &___s_2.UpdatedAt2
					}
				}
			}
		}
	case ENTITY_COL_UPDATED_AT3:
		if s_u3 := s.Upd3; s_u3 != nil {
			if _s_3 := *s_u3; _s_3 != nil {
				return &_s_3.UpdatedAt3
			}
		}
	}
	return nil
}

func gormOrJsonList() []string {
	return []string{EntityGormJsonID, EntityGormJsonName, EntityGormJsonSurname}
}
