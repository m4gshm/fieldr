//go:build fieldr
// +build fieldr

package as_map_no_empty_tag

//go:fieldr -in ../asmap/struct.go -type Struct
//go:generate fieldr -debug -out ../as_map_no_empty_tag-dest/struct_as_map.go -wrap -export -noReceiver -AsMap -AsTagMap -noEmptyTag -Strings -Fields
