package flagx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLookupArgs(t *testing.T) {
	var args = []string{"-run", "abc", "-t", "5s", "-Cool", "-N=1", "-x"}

	v, ok := LookupArgs(args, "run")
	assert.True(t, ok)
	assert.Equal(t, "abc", v)

	v, ok = LookupArgs(args, "t")
	assert.True(t, ok)
	assert.Equal(t, "5s", v)

	v, ok = LookupArgs(args, "Cool")
	assert.True(t, ok)
	assert.Equal(t, "", v)

	v, ok = LookupArgs(args, "N")
	assert.True(t, ok)
	assert.Equal(t, "1", v)

	v, ok = LookupArgs(args, "x")
	assert.True(t, ok)
	assert.Equal(t, "", v)

	v, ok = LookupArgs(args, "???")
	assert.False(t, ok)
	assert.Equal(t, "", v)
}

func TestUnquoteUsage(t *testing.T) {
	type Args struct {
		StringFlag   string        `flag:"StringFlag; def=.*; usage=function name pattern"`
		BoolFlag     bool          `flag:"BoolFlag; def=true; usage=Cool experience"`
		IntFlag      int           `flag:""`
		Int64Flag    int64         `flag:""`
		UintFlag     uint          `flag:"?0;usage=xxx"`
		Uint64Flag   uint64        `flag:"usage=uuu"`
		Float64Flag  float64       `flag:"?1"`
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

func TestNextArgs(t *testing.T) {
	fs := NewFlagSet("non-flag-test1", ContinueOnError)
	runVal := fs.String("run", "", "")
	err := fs.Parse([]string{"-run", "abc", "d", "5s", "--", "-N=1", "-x", "y", "z"})
	assert.NoError(t, err)
	assert.Equal(t, "abc", *runVal)
	assert.Equal(t, 0, fs.NFormalNonFlag())
	assert.Equal(t, []string{"d", "5s", "--", "-N=1", "-x", "y", "z"}, fs.NextArgs())

	fs = NewFlagSet("non-flag-test1", ContinueOnError)
	runVal = fs.String("run", "", "")
	dVal := fs.NonString(0, "", "")
	err = fs.Parse([]string{"-run", "abc", "d", "5s", "--", "-N=1", "-x", "y", "z"})
	assert.NoError(t, err)
	assert.Equal(t, "abc", *runVal)
	assert.Equal(t, "d", *dVal)
	assert.Equal(t, 1, fs.NFormalNonFlag())
	assert.Equal(t, []string{"5s", "--", "-N=1", "-x", "y", "z"}, fs.NextArgs())
}

func TestIndent(t *testing.T) {
	s := "a\nb\n\n"
	r := indent(s, "0")
	assert.Equal(t, "0a\n0b\n0\n", r)
}
