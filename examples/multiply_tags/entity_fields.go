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

func (c EntityCol) field() string { //nolint
	switch c {
	case ENTITY_COL_ID:
		return "BaseEntity.ID"
	case ENTITY_COL_NAME:
		return "Name"
	case ENTITY_COL_SURNAME:
		return "Surname"
	case ENTITY_COL_UPDATED_AT:
		return "Upd.UpdatedAt"
	case ENTITY_COL_UPDATED_AT2:
		return "Upd2.UpdatedAt2"
	case ENTITY_COL_UPDATED_AT3:
		return "Upd3.UpdatedAt3"
	}
	return ""
}

func (s *Entity) val(f EntityCol) interface{} { //nolint
	if s == nil {
		return nil
	}
	switch f {
	case ENTITY_COL_ID:
		if s.BaseEntity != nil {
			return s.BaseEntity.ID
		}
	case ENTITY_COL_NAME:
		return s.Name
	case ENTITY_COL_SURNAME:
		return s.Surname
	case ENTITY_COL_UPDATED_AT:
		if s.Upd != nil && *(s.Upd) != nil && *(*(s.Upd)) != nil && *(*(*(s.Upd))) != nil {
			return (*(*(*(s.Upd)))).UpdatedAt
		}
	case ENTITY_COL_UPDATED_AT2:
		if s.Upd2 != nil && *(s.Upd2) != nil && *(*(s.Upd2)) != nil && *(*(*(s.Upd2))) != nil {
			return (*(*(*(s.Upd2)))).UpdatedAt2
		}
	case ENTITY_COL_UPDATED_AT3:
		if s.Upd3 != nil && *(s.Upd3) != nil {
			return (*(s.Upd3)).UpdatedAt3
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
		if s.BaseEntity != nil {
			return &s.BaseEntity.ID
		}
	case ENTITY_COL_NAME:
		return &s.Name
	case ENTITY_COL_SURNAME:
		return &s.Surname
	case ENTITY_COL_UPDATED_AT:
		if s.Upd != nil && *(s.Upd) != nil && *(*(s.Upd)) != nil && *(*(*(s.Upd))) != nil {
			return &(*(*(*(s.Upd)))).UpdatedAt
		}
	case ENTITY_COL_UPDATED_AT2:
		if s.Upd2 != nil && *(s.Upd2) != nil && *(*(s.Upd2)) != nil && *(*(*(s.Upd2))) != nil {
			return &(*(*(*(s.Upd2)))).UpdatedAt2
		}
	case ENTITY_COL_UPDATED_AT3:
		if s.Upd3 != nil && *(s.Upd3) != nil {
			return &(*(s.Upd3)).UpdatedAt3
		}
	}
	return nil
}

func gormOrJsonList() []string {
	return []string{EntityGormJsonID, EntityGormJsonName, EntityGormJsonSurname}
}
