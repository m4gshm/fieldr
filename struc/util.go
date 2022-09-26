package struc

import "go/types"

func TypeString(typ types.Type, outPkgPath string) string {
	return types.TypeString(typ, basePackQ(outPkgPath))
}

func ObjectString(obj types.Object, outPkgPath string) string {
	return types.ObjectString(obj, basePackQ(outPkgPath))
}

func basePackQ(outPkgPath string) func(p *types.Package) string {
	return func(p *types.Package) string {
		path := p.Path()
		if path == outPkgPath {
			return ""
		}
		return p.Name()
	}
}
