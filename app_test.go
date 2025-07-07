package flagx_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	vd "github.com/bytedance/go-tagexpr/v2/validator"
	"github.com/henrylee2cn/flagx"
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
	loc, _ := time.LoadLocation("Asia/Shanghai") // CST
	t := time.Unix(1581572895, 0).In(loc)
	fmt.Printf(
		"Filter2 start: args=%+v, start at=%v\n",
		c.Args(), t,
	)
	defer func() {
		fmt.Printf(
			"Filter2 end: args=%+v, cost time=%v\n",
			c.Args(), time.Unix(1581572895, 1000).In(loc).Sub(t),
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

func TestCommand(t *testing.T) {
	app := flagx.NewApp()
	app.SetCmdName("testapp")
	app.AddSubaction("a", "subcommand a", new(Action1))
	b := app.AddSubcommand("b", "subcommand b", flagx.FilterFunc(Filter2))
	{
		b.AddSubaction("c", "subcommand c", new(Action2))
		b.AddSubaction("d", "subcommand d", flagx.ActionFunc(Action3))
	}
	app.SetNotFound(func(c *flagx.Context) {
		fmt.Printf("NotFound: args=%+v, path=%q\n", c.Args(), c.CmdPathString())
	})
	assert.NotNil(t, app.LookupSubcommand("a"))
	assert.NotNil(t, app.LookupSubcommand("b"))
	assert.NotNil(t, app.LookupSubcommand("b", "c"))
	assert.NotNil(t, app.LookupSubcommand("b", "d"))
	assert.Nil(t, app.LookupSubcommand("b", "d", "x"))
	assert.Nil(t, app.LookupSubcommand("x"))

	assert.Equal(t, "testapp", app.Root().CmdName())
	assert.Equal(t, "testapp", app.LookupSubcommand("b", "d").Root().CmdName())
	assert.Equal(
		t,
		"$testapp b ...\n"+
			"  subcommand b\n"+
			"$testapp b c\n"+
			"  subcommand c\n"+
			"  -name string\n"+
			"    \tparam name\n"+
			"$testapp b d\n"+
			"  subcommand d\n",
		app.LookupSubcommand("b").UsageText(),
	)
}

func TestScope(t *testing.T) {
	app := flagx.NewApp()
	app.SetScopeMatcher(func(cmdScope, execScope flagx.Scope) error {
		if cmdScope == execScope {
			return nil
		}
		return fmt.Errorf("scopes are not equal: cmdScope=%d, execScope=%d", cmdScope, execScope)
	})
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
	app.AddSubcommand("a", "subcommand a").SetAction(new(Action1), flagx.Scope(1))
	b := app.AddSubcommand("b", "subcommand b", flagx.FilterFunc(Filter2))
	t.Log("scope=0:", app.UsageText(flagx.Scope(0)))

	{
		// cmd: testapp b c
		b.AddSubcommand("c", "subcommand c").SetAction(new(Action2), flagx.Scope(2))
		// cmd: testapp b d
		b.AddSubaction("d", "subcommand d", flagx.ActionFunc(Action3))
	}
	app.SetNotFound(func(c *flagx.Context) {
		fmt.Printf("NotFound: cmd=%q, uasge=%s\n", c.CmdPathString(), c.UsageText())
	})

	// test: testapp
	// not found
	stat := app.Exec(context.TODO(), []string{"-g=flagx", "false"}, flagx.Scope(1))
	assert.True(t, stat.OK())

	// test: testapp a
	stat = app.Exec(context.TODO(), []string{"-g=henry", "true", "a", "-id", "1", "~/m/n"})
	assert.False(t, stat.OK())
	assert.EqualError(t, stat.Cause(), "scopes are not equal: cmdScope=1, execScope=0")

	// test: testapp b
	stat = app.Exec(context.TODO(), []string{"-g=flagx", "false", "b"})
	assert.True(t, stat.OK())

	// test: testapp b c
	// not found
	stat = app.Exec(context.TODO(), []string{"-g=flagx", "false", "b", "c", "name=henry"})
	assert.EqualError(t, stat.Cause(), "scopes are not equal: cmdScope=2, execScope=0")

	// test: testapp b d
	stat = app.Exec(context.TODO(), []string{"-g=flagx", "false", "b", "d"})
	assert.True(t, stat.OK())

	t.Log("no scope:", app.UsageText())
	t.Log("scope=0:", app.UsageText(flagx.Scope(0)))
}
