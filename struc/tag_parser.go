package struc

type TagValueParser = func(tagValue string) TagValue

type TagValueParsers map[TagName]TagValueParser

func (p TagValueParsers) Keys() string {
	result := ""
	for k := range p {
		if len(result) > 0 {
			result += ", "
		}
		result += k
	}
	return result
}

//func NoParser(tagContent string) TagValue {
//	return TagValue(tagContent)
//}
//
//func JSONTagParser(tagContent string) TagValue {
//	omitEmptySuffix := ",omitempty"
//	if strings.HasSuffix(tagContent, omitEmptySuffix) {
//		s := tagContent[0 : len(tagContent)-len(omitEmptySuffix)]
//		return TagValue(s)
//	}
//	return TagValue(tagContent)
//
//}
