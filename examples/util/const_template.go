//go:build fieldr_const_template
// +build fieldr_const_template

package util

const (
	_pk = "{{.FieldTagValue.ID.db}}"

	_selectByID  = "SELECT {{range $i,$tag := .TagValues.db}}{{if gt $i 0}},{{end}}{{$tag}}{{end}} FROM \"" + tableName + "\" WHERE {{.FieldTagValue.ID.db}} = $1"
	_selectByIDs = "SELECT {{range $i,$tag := .TagValues.db}}{{if gt $i 0}},{{end}}{{$tag}}{{end}} FROM \"" + tableName + "\" WHERE {{.FieldTagValue.ID.db}} = ANY($1::int[])"

	_insert = "{{$i:=0}}{{$pk:=.FieldTagValue.ID.db}}INSERT INTO \"" + tableName + "\" ({{range $tag := .TagValues.db}}{{if ne $tag $pk}}{{if gt $i 0}},{{end}}{{$tag}}{{$i = inc $i}}{{end}}{{end}}) " +
		"{{$i:=0}}" +
		"VALUES ({{range $tag := .TagValues.db}}{{if ne $tag $pk}}{{if gt $i 0}},{{end}}{{$i = inc $i}}${{$i}}{{end}}{{end}}) RETURNING {{$pk}}"
	_insertWithID = "INSERT INTO \"" + tableName + "\" ({{range $i,$tag := .TagValues.db}}{{if gt $i 0}},{{end}}{{$tag}}{{end}}) " +
		"VALUES ({{range $i, $tag := .TagValues.db}}{{if gt $i 0}},{{end}}${{inc $i}}{{end}})"

	_upsert = "{{$pk:=.FieldTagValue.ID.db}}INSERT INTO \"" + tableName + "\" ({{range $i,$tag := .TagValues.db}}{{if gt $i 0}},{{end}}{{$tag}}{{end}}) " +
		"VALUES ({{range $i, $tag := .TagValues.db}}{{if gt $i 0}},{{end}}${{inc $i}}{{end}}) ON CONFLICT ({{$pk}}) DO UPDATE SET {{$comma:=false}}{{range $i, $tag := .TagValues.db}}{{if ne $tag $pk}}{{if $comma}},{{end}}{{$tag}}=${{inc $i}}{{$comma = true}}{{end}}{{end}} RETURNING {{$pk}}"
	_updateByID = "{{$pk:=.FieldTagValue.ID.db}}UPDATE \"" + tableName + "\" SET {{$comma:=false}}{{range $i, $tag := .TagValues.db}}{{if ne $tag $pk}}{{if $comma}},{{end}}{{$tag}}=${{inc $i}}{{$comma = true}}{{end}}{{end}} WHERE {{$pk}} = $1"

	_deleteByID  = "DELETE FROM \"" + tableName + "\" WHERE {{.FieldTagValue.ID.db}} = $1"
	_deleteByIDs = "DELETE FROM \"" + tableName + "\" WHERE {{.FieldTagValue.ID.db}} in ($1::int[])"
)
