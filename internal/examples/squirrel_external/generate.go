package squirrel_external

//go:generate fieldr -path ../squirrel -type Entity -out entity_fields.go enum-const -name "join('col', field.name)" -val "tag.db" -type Col -val-access . -ref-access . -list . -flat Versioned -nolint
