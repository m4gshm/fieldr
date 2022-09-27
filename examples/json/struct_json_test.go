package json

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testStruct = Struct[string]{
	MiddleStruct: &MiddleStruct{BaseStruct{&IDAware{ID: 1}}},
	Name:         "NameValue",
	NoJson:       "NoJsonValue",
}

func TestStruct_MarshalJSON(t *testing.T) {

	rawJson, err := testStruct.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	rawExpected, err := json.Marshal(testStruct)

	if err != nil {
		t.Fatal(err)
	}

	expected := string(rawExpected)
	actual := string(rawJson)

	assert.Equal(t, expected, actual)

}
