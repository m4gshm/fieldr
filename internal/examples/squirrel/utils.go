package squirrel

import "unsafe"

func strings(v []Col) []string {
	return *(*[]string)(unsafe.Pointer(&v))
}
