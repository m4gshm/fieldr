//go:build fieldr_const_template
// +build fieldr_const_template

package util

const _pk = "{{.FieldTagValue.ID.db}}"

const _selectByID = "sql_selectByID:SELECT {{range $i,$tag := .TagValues.db}}{{if gt $i 0}},{{end}}{{$tag}}{{end}} FROM " + tableName + " WHERE {{.FieldTagValue.ID.db}} = $1"

const _upsert = "sql_Upsert:{{$pk:=.FieldTagValue.ID.db}}INSERT INTO " + tableName + "({{range $i,$tag := .TagValues.db}}{{if gt $i 0}},{{end}}{{$tag}}{{end}}) " +
	"VALUES ({{range $i, $tag := .TagValues.db}}{{if gt $i 0}},{{end}}${{add $i 1}}{{end}}) DO ON CONFLICT {{$pk}} UPDATE SET {{$comma:=false}}{{range $i, $tag := .TagValues.db}}{{if ne $tag $pk}}{{if $comma}},{{end}}{{$tag}}=${{add $i 1}}{{$comma = true}}{{end}}{{end}} RETURNING {{$pk}}"

const _deleteByID = "sql_deleteByID:DELETE FROM " + tableName + " WHERE {{.FieldTagValue.ID.db}} = $1"
