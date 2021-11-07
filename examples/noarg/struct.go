package noarg

//go:generate fieldr -type Struct -flat Address

type Struct struct {
	ID              int    `toMap:"id"`
	Name            string `toMap:"name"`
	Surname         string `toMap:"surname"`
	noExport        string `toMap:"no_export"` //nolint
	NoTag           string `toMap:""`
	IgnoredInTagMap string
	Address         *Address `toMap:"address"`
}

type Address struct {
	ZipCode     int    `toMap:"zip_code"`
	AddressLine string `toMap:"address_line"`
}
