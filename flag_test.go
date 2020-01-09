package flagx

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestContinueOnUndefined(t *testing.T) {
	var args = []string{"test", "-x=1", "-y"}
	fs := NewFlagSet(args[0], ContinueOnError)
	fs.String("x", "", "")
	err := fs.Parse(args[1:])
	assert.EqualError(t, err, "flag provided but not defined: -y")
	fs.Usage()

	fs = NewFlagSet(args[0], ContinueOnError|ContinueOnUndefined)
	x := fs.String("x", "", "")
	err = fs.Parse(args[1:])
	assert.NoError(t, err)
	assert.Equal(t, "1", *x)
}

func ExampleStructVars() {
	os.Args = []string{"go test", "-test.timeout", "30s", "-test.v", "-test.count", "1", "-test.run", "^(TestStructVars)$"}
	type Args struct {
		Run     string        `flag:"test.run; def=.*; usage=function name pattern"`
		Timeout time.Duration `flag:"test.timeout"`
		V       bool          `flag:"test.v"`
		X       int           `flag:"def=10"`
	}
	var args Args
	err := StructVars(&args)
	if err != nil {
		panic(err)
	}
	Parse()
	fmt.Printf("%+v\n", args)
	// Output:
	// {Run:^(TestStructVars)$ Timeout:30s V:true X:10}
}

func ExampleMoreStructVars() {
	type Args struct {
		Run     string        `flag:"run; def=.*; usage=function name pattern"`
		Timeout time.Duration `flag:"timeout,t"`
		Cool    bool          `flag:"usage=Cool experience"`
		View    bool          `flag:"view,v; def=true"`
		N       int           `flag:""`
	}
	for i, a := range [][]string{
		{}, // test default value
		{"-run", "abc", "-timeout", "5s", "-Cool", "-N", "1"},
		{"-run", "abc", "-t", "5s", "-Cool", "-N", "1"},
		{"-run", "", "-t", "0", "-N", "0"},                            // test zero value
		{"-run", "", "-t", "0", "-x", "-N", "10", "-y", "z"},          // test zero value and ContinueOnUndefined
		{"-run", "", "-t", "0", "-x", "-N", "10", "-y", "z", "extra"}, // test extra and ContinueOnUndefined
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
	// {Run:abc Timeout:5s Cool:true View:true N:1}
	// {Run:abc Timeout:5s Cool:true View:true N:1}
	// {Run: Timeout:0s Cool:false View:true N:0}
	// {Run: Timeout:0s Cool:false View:true N:10}
	// {Run: Timeout:0s Cool:false View:true N:10}
}

func TestTidyArgs(t *testing.T) {
	for i, a := range [][]string{
		{}, // test default value
		{"-run", "abc", "-timeout", "5s", "-Cool", "-N", "1"},
		{"-run", "abc", "-t", "5s", "-Cool", "-N", "1"},
		{"-run", "", "-t", "0", "-N", "0"},
		{"-run", "", "-t", "0", "-x", "-N", "0", "-y", "z"},
		{"-run", "", "m"},
	} {
		tidiedArgs, lastArgs, err := tidyArgs(a, func(string) (want bool, next bool) { return true, true })
		assert.NoError(t, err)
		switch i {
		case 0, 1, 2, 3:
			assert.Equal(t, []string{}, lastArgs)
		case 5:
			assert.Equal(t, []string{"m"}, lastArgs)
		}
		t.Logf("i:%d, tidiedArgs:%#v", i, tidiedArgs)
	}
	args := []string{"-run", "abc", "--", "-c", "2"}
	tidiedArgs, args, err := tidyArgs(args, func(string) (want bool, next bool) { return true, true })
	assert.NoError(t, err)
	assert.Equal(t, []string{"-run", "abc"}, tidiedArgs)
	assert.Equal(t, []string{"-c", "2"}, args)
	tidiedArgs, args, err = tidyArgs(args, func(string) (want bool, next bool) { return true, true })
	assert.NoError(t, err)
	assert.Equal(t, []string{"-c", "2"}, tidiedArgs)
	assert.Equal(t, []string{}, args)
}
