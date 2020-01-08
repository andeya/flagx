package flagx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLookup(t *testing.T) {
	var args = []string{"-run", "abc", "-t", "5s", "-Cool", "true", "-v", "false", "-N", "1"}
	v, ok := Lookup(args, "Cool")
	assert.True(t, ok)
	assert.Equal(t, "true", v)
	v, ok = Lookup(args, "???")
	assert.False(t, ok)
	assert.Equal(t, "", v)
}

func TestUnquoteUsage(t *testing.T) {
	type Args struct {
		StringFlag   string        `flag:"StringFlag; def=.*; usage=function name pattern"`
		BoolFlag     bool          `flag:"BoolFlag; def=true; usage=Cool experience"`
		IntFlag      int           `flag:""`
		Int64Flag    int64         `flag:""`
		UintFlag     uint          `flag:""`
		Uint64Flag   uint64        `flag:""`
		Float64Flag  float64       `flag:""`
		DurationFlag time.Duration `flag:""`
	}
	var args Args
	fs := NewFlagSet("TestUnquoteUsage", 0)
	err := fs.StructVars(&args)
	assert.NoError(t, err)
	fs.VisitAll(func(f *Flag) {
		name, usage := UnquoteUsage(f)
		t.Logf("name:%q, usage:%q", name, usage)
	})
	fs.Usage()
}
