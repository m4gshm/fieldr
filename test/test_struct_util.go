// Code generated by 'const -type TestStruct -wrap -export -output test_struct_util.go'; DO NOT EDIT.

package test

type (
	TestStructField     string
	TestStructFields    []TestStructField
	TestStructTag       string
	TestStructTags      []TestStructTag
	TestStructTagValue  string
	TestStructTagValues []TestStructTagValue
)

const (
	TestStruct_ID     = TestStructField("ID")
	TestStruct_Name   = TestStructField("Name")
	TestStruct_NoJson = TestStructField("NoJson")
	TestStruct_ts     = TestStructField("ts")

	TestStruct_db   = TestStructTag("db")
	TestStruct_json = TestStructTag("json")

	TestStruct_db_ID     = TestStructTagValue("ID")
	TestStruct_db_Name   = TestStructTagValue("NAME")
	TestStruct_db_NoJson = TestStructTagValue("NO_JSON")
	TestStruct_db_ts     = TestStructTagValue("TS")

	TestStruct_json_ID   = TestStructTagValue("id")
	TestStruct_json_Name = TestStructTagValue("name,omitempty")
	TestStruct_json_ts   = TestStructTagValue("ts")
)

var (
	testStruct_Fields = TestStructFields{TestStruct_ID, TestStruct_Name, TestStruct_NoJson, TestStruct_ts}

	testStruct_Tags = TestStructTags{TestStruct_db, TestStruct_json}

	testStruct_Field_Tags = map[TestStructField]TestStructTags{
		TestStruct_ID:     TestStructTags{TestStruct_db, TestStruct_json},
		TestStruct_Name:   TestStructTags{TestStruct_db, TestStruct_json},
		TestStruct_NoJson: TestStructTags{TestStruct_db},
		TestStruct_ts:     TestStructTags{TestStruct_db, TestStruct_json},
	}

	testStruct_Tag_Values = map[TestStructTag]TestStructTagValues{
		TestStruct_db:   TestStructTagValues{TestStruct_db_ID, TestStruct_db_Name, TestStruct_db_NoJson, TestStruct_db_ts},
		TestStruct_json: TestStructTagValues{TestStruct_json_ID, TestStruct_json_Name, TestStruct_json_ts},
	}

	testStruct_Tag_Fields = map[TestStructTag]TestStructFields{
		TestStruct_db:   TestStructFields{TestStruct_ID, TestStruct_Name, TestStruct_NoJson, TestStruct_ts},
		TestStruct_json: TestStructFields{TestStruct_ID, TestStruct_Name, TestStruct_ts},
	}

	testStruct_Field_Tag_Value = map[TestStructField]map[TestStructTag]TestStructTagValue{
		TestStruct_ID:     map[TestStructTag]TestStructTagValue{TestStruct_db: TestStruct_db_ID, TestStruct_json: TestStruct_json_ID},
		TestStruct_Name:   map[TestStructTag]TestStructTagValue{TestStruct_db: TestStruct_db_Name, TestStruct_json: TestStruct_json_Name},
		TestStruct_NoJson: map[TestStructTag]TestStructTagValue{TestStruct_db: TestStruct_db_NoJson},
		TestStruct_ts:     map[TestStructTag]TestStructTagValue{TestStruct_db: TestStruct_db_ts, TestStruct_json: TestStruct_json_ts},
	}
)

func (v TestStructFields) Strings() []string {
	strings := make([]string, 0, len(v))
	for i, v := range v {
		strings[i] = string(v)
	}
	return strings
}

func (v TestStructTags) Strings() []string {
	strings := make([]string, 0, len(v))
	for i, v := range v {
		strings[i] = string(v)
	}
	return strings
}

func (v TestStructTagValues) Strings() []string {
	strings := make([]string, 0, len(v))
	for i, v := range v {
		strings[i] = string(v)
	}
	return strings
}

func (v *TestStruct) FieldValue(field TestStructField) interface{} {
	switch field {
	case TestStruct_ID:
		return v.ID
	case TestStruct_Name:
		return v.Name
	case TestStruct_NoJson:
		return v.NoJson
	case TestStruct_ts:
		return v.ts
	}
	return nil
}

func (v *TestStruct) FieldValueByTagValue(tag TestStructTagValue) interface{} {
	switch tag {
	case TestStruct_db_ID, TestStruct_json_ID:
		return v.ID
	case TestStruct_db_Name, TestStruct_json_Name:
		return v.Name
	case TestStruct_db_NoJson:
		return v.NoJson
	case TestStruct_db_ts, TestStruct_json_ts:
		return v.ts
	}
	return nil
}

func (v *TestStruct) AsMap() map[TestStructField]interface{} {
	return map[TestStructField]interface{}{
		TestStruct_ID:     v.ID,
		TestStruct_Name:   v.Name,
		TestStruct_NoJson: v.NoJson,
		TestStruct_ts:     v.ts,
	}
}