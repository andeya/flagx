package flagx

import (
	"fmt"
	"os"
	"strconv"
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
	fs.Usage()

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

func ExampleStructVars() {
	type Args struct {
		Run     string        `flag:"run; def=.*; usage=function name pattern"`
		Timeout time.Duration `flag:"timeout,t"`
		Cool    bool          `flag:"usage=Cool experience"`
		View    bool          `flag:"view,v; def=true"`
		N       int           `flag:""`
	}
	for i, a := range [][]string{
		{}, // test default value
		{"-run", "abc", "-timeout", "5s", "-Cool", "true", "-view", "false", "-N", "1"},
		{"-run", "abc", "-t", "5s", "-Cool", "true", "-v", "false", "-N", "1"},
		{"-run", "-t", "-Cool", "-v", "-N"}, // test without value
	} {
		var args Args
		fs := NewFlagSet(strconv.Itoa(i), ContinueOnError|ContinueOnUndefined)
		err := fs.StructVars(&args)
		if err != nil {
			panic(err)
		}
		err = fs.Parse(a)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%+v\n", args)
		fs.Usage()
	}
	// Output:
	// {Run:.* Timeout:0s Cool:false View:true N:0}
	// {Run:abc Timeout:5s Cool:true View:false N:1}
	// {Run:abc Timeout:5s Cool:true View:false N:1}
	// {Run: Timeout:0s Cool:true View:true N:0}
}
