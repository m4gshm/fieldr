// Code generated by 'fieldr'; DO NOT EDIT.

package gorm

type EntityCol string

const (
	ENTITY_COL_ID           EntityCol = "ID"
	ENTITY_COL_UPDATED_AT   EntityCol = "UPDATED_AT"
	ENTITY_COL_NAME         EntityCol = "NAME"
	ENTITY_COL_SURNAME      EntityCol = "SURNAME"
	EntityGormID                      = "ID"
	EntityGormUpdatedAt               = "UPDATED_AT"
	EntityGormName                    = "NAME"
	EntityGormSurname                 = "SURNAME"
	EntityJsonID                      = "id"
	EntityJsonUpdatedAt               = "updateAt"
	EntityJsonName                    = "name"
	EntityGormJsonID                  = "id"
	EntityGormJsonUpdatedAt           = "updateAt"
	EntityGormJsonName                = "NAME"
	EntityGormJsonSurname             = "SURNAME"
)

func entityCols() []EntityCol {
	return []EntityCol{
		ENTITY_COL_ID,
		ENTITY_COL_UPDATED_AT,
		ENTITY_COL_NAME,
		ENTITY_COL_SURNAME,
	}
}

func (c EntityCol) field() string {
	switch c {
	case ENTITY_COL_ID:
		return "ID"
	case ENTITY_COL_UPDATED_AT:
		return "UpdatedAt"
	case ENTITY_COL_NAME:
		return "Name"
	case ENTITY_COL_SURNAME:
		return "Surname"
	}
	return ""
}

func (c EntityCol) val(s *Entity) interface{} {
	switch c {
	case ENTITY_COL_ID:
		return s.ID
	case ENTITY_COL_UPDATED_AT:
		return s.UpdatedAt
	case ENTITY_COL_NAME:
		return s.Name
	case ENTITY_COL_SURNAME:
		return s.Surname
	}
	return nil
}

func (c EntityCol) ref(s *Entity) interface{} {
	switch c {
	case ENTITY_COL_ID:
		return &s.ID
	case ENTITY_COL_UPDATED_AT:
		return &s.UpdatedAt
	case ENTITY_COL_NAME:
		return &s.Name
	case ENTITY_COL_SURNAME:
		return &s.Surname
	}
	return nil
}

func gormOrJsonList() []string {
	return []string{
		EntityGormJsonID,
		EntityGormJsonUpdatedAt,
		EntityGormJsonName,
		EntityGormJsonSurname,
	}
}
