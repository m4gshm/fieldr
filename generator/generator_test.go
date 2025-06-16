package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_IdentName(t *testing.T) {
	assert.Equal(t, "IDMain", IdentName("idMain", true))
	assert.Equal(t, "idMain", IdentName("iDMain", false))

	assert.Equal(t, "URLmain", IdentName("urlmain", true))
	assert.Equal(t, "urlMain", IdentName("UrLMain", false))

	assert.Equal(t, "sETName", IdentName("SETName", false))
}

func Test_ArgName(t *testing.T) {
	assert.Equal(t, "", ArgName(""))
	assert.Equal(t, "abcd", ArgName("ABCD"))
	assert.Equal(t, "IDMain", ArgName("IDMain"))
	assert.Equal(t, "idMain", ArgName("IdMain"))
	assert.Equal(t, "d", ArgName("D"))
	assert.Equal(t, "htTP", ArgName("HtTP"))
}
