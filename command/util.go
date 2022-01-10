package command

func toSet(values []string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, v := range values {
		set[v] = struct{}{}
	}
	return set
}
