package test

type TestStruct struct {
	ID   int    `db:"_id" json:"ID"`
	Name string `db:"_name"`
}
