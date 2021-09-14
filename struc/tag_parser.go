package struc

import (
	"log"
	"regexp"
	"strings"
)

type TagValueParser = func(tagValue string) TagValue
type TagValueParsers map[TagName]TagValueParser

func (p TagValueParsers) Keys() string {
	result := ""
	for k, _ := range p {
		if len(result) > 0 {
			result += ", "
		}
		result += string(k)
	}
	return result
}

func NoParser(tagContent string) TagValue {
	return TagValue(tagContent)
}

func JsonTagParser(tagContent string) TagValue {
	omitEmptySuffix := ",omitempty"
	if strings.HasSuffix(tagContent, omitEmptySuffix) {
		s := tagContent[0 : len(tagContent)-len(omitEmptySuffix)]
		return TagValue(s)
	}
	return TagValue(tagContent)

}

func RegExpParser(regExpr string) TagValueParser {
	pattern, err := regexp.Compile(regExpr)
	if err != nil {
		log.Fatal(err)
	}

	return func(tagContent string) TagValue {
		return TagValue(extractTagValue(string(tagContent), pattern))
	}
}
