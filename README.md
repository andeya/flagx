# flagx [![report card](https://goreportcard.com/badge/github.com/andeya/flagx?style=flat-square)](http://goreportcard.com/report/andeya/flagx) [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/andeya/flagx)

Standard flag package extension with more features, such as struct flag, app framework, etc.

## Extension Feature

- Add `const ContinueOnUndefined ErrorHandling`: ignore provided but undefined flags
- Add `*FlagSet.StructVars`: define flags based on struct tags and bind to fields
  - The list of supported types is consistent with the standard package:
    - `string`
    - `bool`
    - `int`
    - `int64`
    - `uint`
    - `uint64`
    - `float64`
    - `time.Duration`
- Add `LookupArgs`: lookup the value corresponding to a name directly from arguments
- Provide application framework
- Support define non-flag
    - Use `?{index}` (such as `?0`, `?1`, `?2`) in struct tag to define non-flag
- For more features, please open the issue

## Test Demo

- Ignore provided but undefined flags

```go
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
```

- Define flags based on struct tags and bind to fields

```go
func ExampleStructVars() {
	os.Args = []string{
		"go test",
		"-test.timeout", "30s",
		"-test.v",
		"-test.count", "1",
		"-test.run", "^(TestStructVars)$",
		"flag_test.go",
	}
	type Args struct {
		Run     string        `flag:"test.run; def=.*; usage=function name pattern"`
		Timeout time.Duration `flag:"test.timeout"`
		V       bool          `flag:"test.v"`
		X       int           `flag:"def=10"`
		Y       string        `flag:"?0"` // the first non-flag
	}
	var args Args
	err := StructVars(&args)
	if err != nil {
		panic(err)
	}
	Parse()
	fmt.Printf("%+v\n", args)
	// Output:
	// {Run:^(TestStructVars)$ Timeout:30s V:true X:10 Y:flag_test.go}
}
```

- Lookup the value corresponding to a name directly from arguments

```go
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
```

- Aapplication

```go
package flagx_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	vd "github.com/bytedance/go-tagexpr/v2/validator"
	"github.com/andeya/flagx"
	"github.com/stretchr/testify/assert"
)

func ExampleApp() {
	app := flagx.NewApp()
	app.SetCmdName("testapp")
	app.SetDescription("this is a app for testing")
	app.SetAuthors([]flagx.Author{{
		Name:  "henrylee2cn",
		Email: "henrylee2cn@gmail.com",
	}})
	app.SetValidator(func(v interface{}) error {
		return vd.Validate(v)
	})
	app.AddFilter(new(Filter1))
	// cmd: testapp a
	app.AddSubaction("a", "subcommand a", new(Action1))
	b := app.AddSubcommand("b", "subcommand b", flagx.FilterFunc(Filter2))
	{
		// cmd: testapp b c
		b.AddSubaction("c", "subcommand c", new(Action2))
		// cmd: testapp b d
		b.AddSubaction("d", "subcommand d", flagx.ActionFunc(Action3))
	}
	app.SetNotFound(func(c *flagx.Context) {
		fmt.Printf("NotFound: cmd=%q, uasge=%s\n", c.CmdPathString(), c.UsageText())
	})

	fmt.Println(app.UsageText())

	// test: testapp
	// not found
	stat := app.Exec(context.TODO(), []string{"-g=flagx", "false"})
	if !stat.OK() {
		panic(stat)
	}

	// test: testapp a
	stat = app.Exec(context.TODO(), []string{"-g=henry", "true", "a", "-id", "1", "~/m/n"})
	if !stat.OK() {
		panic(stat)
	}

	// test: testapp b
	stat = app.Exec(context.TODO(), []string{"-g=flagx", "false", "b"})
	if !stat.OK() {
		panic(stat)
	}

	// test: testapp b c
	// not found
	stat = app.Exec(context.TODO(), []string{"-g=flagx", "false", "b", "c", "name=henry"})
	if !stat.OK() {
		panic(stat)
	}

	// test: testapp b d
	stat = app.Exec(context.TODO(), []string{"-g=flagx", "false", "b", "d"})
	if !stat.OK() {
		panic(stat)
	}

	// Output:
	// testapp - v0.0.1
	//
	// this is a app for testing
	//
	// USAGE:
	//   -g string
	//     	global param g
	//   ?0 bool
	//     	param view
	//   $testapp a
	//     subcommand a
	//     -id int
	//       	param id
	//     ?0 string
	//       	param path
	//   $testapp b ...
	//     subcommand b
	//   $testapp b c
	//     subcommand c
	//     -name string
	//       	param name
	//   $testapp b d
	//     subcommand d
	//
	// AUTHOR:
	//   henrylee2cn <henrylee2cn@gmail.com>
	//
	// NotFound: cmd="testapp", uasge=-g string
	//   	global param g
	// ?0 bool
	//   	param view
	// $testapp a
	//   subcommand a
	//   -id int
	//     	param id
	//   ?0 string
	//     	param path
	// $testapp b ...
	//   subcommand b
	// $testapp b c
	//   subcommand c
	//   -name string
	//     	param name
	// $testapp b d
	//   subcommand d
	//
	// Filter1 start: args=[-g=henry true a -id 1 ~/m/n], G=henry
	// Action1: args=[-g=henry true a -id 1 ~/m/n], path="testapp a", object=&{ID:1 Path:~/m/n}
	// Filter1 end: args=[-g=henry true a -id 1 ~/m/n]
	// NotFound: cmd="testapp b", uasge=$testapp b ...
	//   subcommand b
	// $testapp b c
	//   subcommand c
	//   -name string
	//     	param name
	// $testapp b d
	//   subcommand d
	//
	// Filter1 start: args=[-g=flagx false b c name=henry], V=false
	// Filter2 start: args=[-g=flagx false b c name=henry], start at=2020-02-13 13:48:15 +0800 CST
	// Action2: args=[-g=flagx false b c name=henry], path="testapp b c", object=&{Name:}
	// Filter2 end: args=[-g=flagx false b c name=henry], cost time=1µs
	// Filter1 end: args=[-g=flagx false b c name=henry]
	// Filter1 start: args=[-g=flagx false b d], V=false
	// Filter2 start: args=[-g=flagx false b d], start at=2020-02-13 13:48:15 +0800 CST
	// Action3: args=[-g=flagx false b d], path="testapp b d"
	// Filter2 end: args=[-g=flagx false b d], cost time=1µs
	// Filter1 end: args=[-g=flagx false b d]
}

type Filter1 struct {
	G string `flag:"g;usage=global param g"`
	V bool   `flag:"?0;usage=param view"`
}

func (f *Filter1) Filter(c *flagx.Context, next flagx.ActionFunc) {
	if f.V {
		fmt.Printf("Filter1 start: args=%+v, G=%s\n", c.Args(), f.G)
	} else {
		fmt.Printf("Filter1 start: args=%+v, V=%v\n", c.Args(), f.V)
	}
	defer fmt.Printf("Filter1 end: args=%+v\n", c.Args())
	next(c)
}

func Filter2(c *flagx.Context, next flagx.ActionFunc) {
	t := time.Unix(1581572895, 0)
	fmt.Printf(
		"Filter2 start: args=%+v, start at=%v\n",
		c.Args(), t,
	)
	defer func() {
		fmt.Printf(
			"Filter2 end: args=%+v, cost time=%v\n",
			c.Args(), time.Unix(1581572895, 1000).Sub(t),
		)
	}()
	next(c)
}

type Action1 struct {
	ID   int    `flag:"id;usage=param id" vd:"@:$!=0; msg:'empty ID'"`
	Path string `flag:"?0;usage=param path"`
}

func (a *Action1) Execute(c *flagx.Context) {
	fmt.Printf("Action1: args=%+v, path=%q, object=%+v\n", c.Args(), c.CmdPathString(), a)
}

type Action2 struct {
	Name string `flag:"name;usage=param name"`
}

func (a *Action2) Execute(c *flagx.Context) {
	fmt.Printf("Action2: args=%+v, path=%q, object=%+v\n", c.Args(), c.CmdPathString(), a)
}

func Action3(c *flagx.Context) {
	fmt.Printf("Action3: args=%+v, path=%q\n", c.Args(), c.CmdPathString())
}
```
