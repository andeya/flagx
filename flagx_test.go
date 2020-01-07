package flagx

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContinueOnUndefined(t *testing.T) {
	fs := NewFlagSet(os.Args[0], ContinueOnError)
	run := fs.String("test.run", "", "")
	err := fs.Parse(os.Args[1:])
	assert.NotNil(t, err)
	t.Log(err)

	fs = NewFlagSet(os.Args[0], ContinueOnError|ContinueOnUndefined)
	run = fs.String("test.run", "", "")
	err = fs.Parse(os.Args[1:])
	assert.NoError(t, err)
	assert.True(t, strings.Contains(*run, "TestContinueOnUndefined"))
}
