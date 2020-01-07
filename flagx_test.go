package flagx

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestContinueOnUndefined(t *testing.T) {
	fs := NewFlagSet(os.Args[0], ContinueOnError)
	run := fs.String("test.run", "", "")
	err := fs.Parse(os.Args[1:])
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "flag provided but not defined:"))

	fs = NewFlagSet(os.Args[0], ContinueOnError|ContinueOnUndefined)
	run = fs.String("test.run", "", "")
	err = fs.Parse(os.Args[1:])
	assert.NoError(t, err)
	assert.True(t, strings.Contains(*run, "TestContinueOnUndefined"))
}

func TestStructVars(t *testing.T) {
	type Args struct {
		Run     string        `flag:"test.run; def=.*; usage=function name pattern"`
		Timeout time.Duration `flag:"test.timeout"`
	}
	var args Args
	err := StructVars(&args)
	assert.NoError(t, err)
	Parse()
	assert.NoError(t, err)
	assert.True(t, strings.Contains(args.Run, "TestStructVars"))
	t.Logf("%+v", args)
	PrintDefaults()
}
