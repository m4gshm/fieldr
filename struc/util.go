package struc

import "go/types"

func TypeString(typ types.Type, basePackageName string) string {
	ts := types.TypeString(typ, func(p *types.Package) string {
		n := p.Name()
		if n == basePackageName {
			return ""
		}
		return n
	})
	return ts
}
