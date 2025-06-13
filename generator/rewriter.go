package generator

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/m4gshm/fieldr/logger"
	"github.com/m4gshm/fieldr/model/struc"
)

type RewriteTrigger string

const (
	RewriteTriggerEmpty RewriteTrigger = ""
	RewriteTriggerField RewriteTrigger = "field"
	RewriteTriggerType  RewriteTrigger = "type"
)

type RewriteEngine string

const (
	RewriteEngineFmt RewriteEngine = "fmt"
)

type CodeRewriter struct {
	byFieldName map[struc.FieldName][]func(string) string
	byFieldType map[string][]func(string) string
	all         []func(string) string
}

func NewCodeRewriter(fieldValueRewriters []string) (*CodeRewriter, error) {
	r := &CodeRewriter{
		byFieldName: map[string][]func(string) string{},
		byFieldType: map[string][]func(string) string{},
		all:         []func(string) string{},
	}
	for _, rewList := range fieldValueRewriters {
		rewritersCfg := strings.Split(rewList, struc.ListValuesSeparator)
		for _, rewriterCfg := range rewritersCfg {
			var (
				rewParts        = strings.Split(rewriterCfg, struc.KeyValueSeparator)
				rewTrigger      RewriteTrigger
				rewTriggerValue string
				rewEngingCfg    string
			)
			if len(rewParts) == 1 {
				rewTrigger = RewriteTriggerEmpty
				rewTriggerValue = rewParts[0]
				rewEngingCfg = rewTriggerValue
			} else if len(rewParts) == 2 {
				rewTrigger = RewriteTriggerField
				rewTriggerValue = rewParts[0]
				rewEngingCfg = rewParts[1]
			} else if len(rewParts) == 3 {
				rewTrigger = RewriteTrigger(rewParts[0])
				rewTriggerValue = rewParts[1]
				rewEngingCfg = rewParts[2]
			} else {
				return nil, errors.Errorf("Unsupported transformValue format '%v'", rewriterCfg)
			}

			var (
				rewEngineParts = strings.Split(rewEngingCfg, struc.ReplaceableValueSeparator)
				rewEngine      RewriteEngine
				rewEngineData  string
			)
			if len(rewEngineParts) == 0 {
				return nil, errors.Errorf("Undefined rewriter value '%v'", rewriterCfg)
			} else if len(rewEngineParts) == 2 {
				rewEngine = RewriteEngine(rewEngineParts[0])
				rewEngineData = rewEngineParts[1]
			} else {
				return nil, errors.Errorf("Unsupported rewriter value '%v' from '%v'", rewEngineParts[0], rewEngingCfg)
			}

			var rewFunc func(string) string
			switch rewEngine {
			case RewriteEngineFmt:
				rewFunc = func(fieldValue string) string {
					return fmt.Sprintf(rewEngineData, fieldValue)
				}
			default:
				return nil, errors.Errorf("Unsupported transform engine '%v' from '%v'", rewEngine, rewriterCfg)
			}

			switch rewTrigger {
			case RewriteTriggerEmpty:
				r.all = append(r.all, rewFunc)
			case RewriteTriggerField:
				r.byFieldName[rewTriggerValue] = append(r.byFieldName[rewTriggerValue], rewFunc)
			case RewriteTriggerType:
				r.byFieldType[rewTriggerValue] = append(r.byFieldType[rewTriggerValue], rewFunc)
			default:
				return nil, errors.Errorf("Unsupported transform trigger '%v' from '%v'", rewTrigger, rewriterCfg)
			}
		}
	}
	return r, nil
}

func (rewrite *CodeRewriter) Transform(fieldName string, filedType string, fieldRef string) (string, bool) {
	rewriters, ok := rewrite.byFieldName[fieldName]
	if !ok {
		
		if rewriters, ok = rewrite.byFieldType[filedType]; !ok {
			logger.Debugf("no rewriter by type for field %s, type %s", fieldName, filedType)
			rewriters = rewrite.all[:]
		}
	}
	if len(rewriters) == 0 {
		return fieldRef, false
	}
	rewrited := false
	for _, rewrite := range rewriters {
		before := fieldRef
		fieldRef = rewrite(fieldRef)
		logger.Debugf("transforming field value: field %s, value before %s, after %s", fieldName, before, fieldRef)
		rewrited = rewrited || before != fieldRef
	}
	return fieldRef, rewrited
}
