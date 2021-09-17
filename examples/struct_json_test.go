package examples

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var testStruct = Struct{
	ID:     1,
	Name:   "NameValue",
	NoJson: "NoJsonValue",
	ts:     time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC),
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
