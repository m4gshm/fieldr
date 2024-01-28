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
		ENTITY_COL_UPDATED_AT3}
}

func (s *Entity) val(f EntityCol) interface{} { //nolint
	if s == nil {
		return nil
	}
	switch f {
	case ENTITY_COL_ID:
		if be := s.BaseEntity; be != nil {
			return be.ID
		}
	case ENTITY_COL_NAME:
		return s.Name
	case ENTITY_COL_SURNAME:
		return s.Surname
	case ENTITY_COL_UPDATED_AT:
		if u := s.Upd; u != nil {
			if _u := *u; _u != nil {
				if _r_u := *_u; _r_u != nil {
					if _r_r_u := *_r_u; _r_r_u != nil {
						return _r_r_u.UpdatedAt
					}
				}
			}
		}
	case ENTITY_COL_UPDATED_AT2:
		if u2 := s.Upd2; u2 != nil {
			if _u2 := *u2; _u2 != nil {
				if _r_u2 := *_u2; _r_u2 != nil {
					if _r_r_u2 := *_r_u2; _r_r_u2 != nil {
						return _r_r_u2.UpdatedAt2
					}
				}
			}
		}
	case ENTITY_COL_UPDATED_AT3:
		if u3 := s.Upd3; u3 != nil {
			if _u3 := *u3; _u3 != nil {
				return _u3.UpdatedAt3
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
		if be := s.BaseEntity; be != nil {
			return &be.ID
		}
	case ENTITY_COL_NAME:
		return &s.Name
	case ENTITY_COL_SURNAME:
		return &s.Surname
	case ENTITY_COL_UPDATED_AT:
		if u := s.Upd; u != nil {
			if _u := *u; _u != nil {
				if _r_u := *_u; _r_u != nil {
					if _r_r_u := *_r_u; _r_r_u != nil {
						return &_r_r_u.UpdatedAt
					}
				}
			}
		}
	case ENTITY_COL_UPDATED_AT2:
		if u2 := s.Upd2; u2 != nil {
			if _u2 := *u2; _u2 != nil {
				if _r_u2 := *_u2; _r_u2 != nil {
					if _r_r_u2 := *_r_u2; _r_r_u2 != nil {
						return &_r_r_u2.UpdatedAt2
					}
				}
			}
		}
	case ENTITY_COL_UPDATED_AT3:
		if u3 := s.Upd3; u3 != nil {
			if _u3 := *u3; _u3 != nil {
				return &_u3.UpdatedAt3
			}
		}
	}
	return nil
}

func gormOrJsonList() []string {
	return []string{EntityGormJsonID, EntityGormJsonName, EntityGormJsonSurname}
}
