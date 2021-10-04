package noarg

//go:generate fieldr -type StructWrap -wrap -hardcode -compact

type StructWrap struct {
	ID              int    `toMap:"id"`
	Name            string `toMap:"name"`
	Surname         string `toMap:"surname"`
	noExport        string `toMap:"no_export"` //nolint
	NoTag           string `toMap:""`
	IgnoredInTagMap string
}
