package generator

import (
	"go/token"
	"runtime"
	"testing"

	"github.com/m4gshm/fieldr/logger"
	"github.com/m4gshm/fieldr/model/util"
	"github.com/stretchr/testify/assert"
)

func init() {
	logger.Init(false)
}

func Test_NexExpr(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)

	fileSet := token.NewFileSet()
	pkgs, err := util.ExtractPackages(fileSet, nil, filename)
	assert.NoError(t, err)

	typ, _, _, err := util.FindTypePackageFile("rrs", fileSet, pkgs)
	assert.NoError(t, err)

	res := generateNewObjectExpr(typ, "", "s")
	assert.Equal(t, `s1 := new(string)
s0 := &s1
s = &s0`, res)
}

type rs = *string
type rrs = *rs
