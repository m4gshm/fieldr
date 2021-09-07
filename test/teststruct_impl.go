package test

func (t *TestStruct) impl() string {
	return string(TestStruct_db_Name) + " " + string(TestStruct_db_ID)
}
