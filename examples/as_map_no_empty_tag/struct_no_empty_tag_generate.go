//go:build fieldr
// +build fieldr

package as_map_no_empty_tag

//go:fieldr -in ../as_map/struct.go -type Struct
//go:generate fieldr -debug -out struct_as_map.go -wrap -export -noReceiver -AsMap -AsTagMap -noEmptyTag
