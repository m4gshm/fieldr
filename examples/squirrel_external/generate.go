package squirrel_external

//go:generate fieldr -package ../squirrel -type Entity -out entity_fields.go enum-const -name "{{ join \"col\" field.name }}" -val "tag.db" -type Col -val-accessor -ref-accessor -flat Versioned
