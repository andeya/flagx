package flagx_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/henrylee2cn/flagx"
	"github.com/stretchr/testify/assert"
)

func ExampleApp() {
	app := flagx.NewApp()
	app.SetName("TestApp")
	app.SetCmdName("testapp")
	app.SetDescription("this is a app for testing")
	app.SetAuthors([]flagx.Author{{
		Name:  "henrylee2cn",
		Email: "henrylee2cn@gmail.com",
	}})
	date, _ := time.Parse(time.RFC3339, "2020-01-10T15:17:03+08:00")
	app.SetCompiled(date)
	app.Use(Mw2)
	app.SetOptions(new(GlobalHandler))
	app.SetNotFound(func(c *flagx.Context) {
		argsInfo := c.ArgsInfo()
		fmt.Printf(
			"Not Found, args: cmd=%q, options=%v\n",
			argsInfo.Command, argsInfo.Options,
		)
	})
	app.MustAddAction("a", "test-a", new(AHandler))
	app.MustAddAction("c", "test-c", flagx.HandlerFunc(CHandler))

	stat := app.Exec(context.TODO(), []string{"a", "-a", "x"})
	if !stat.OK() {
		panic(stat)
	}

	stat = app.Exec(context.TODO(), []string{"c"})
	if !stat.OK() {
		panic(stat)
	}

	stat = app.Exec(context.TODO(), []string{"-g", "g0", "--", "c"})
	if !stat.OK() {
		panic(stat)
	}

	stat = app.Exec(context.TODO(), []string{"b", "-no"})
	if !stat.OK() {
		panic(stat)
	}

	// Output:
	// Mw2: cmd="", options=[] start
	// AHandler cmd="a", options=[-a x], -a=x
	// Mw2: cmd="", options=[] end
	// Mw2: cmd="", options=[] start
	// CHandler cmd="c", options=[]
	// Mw2: cmd="", options=[] end
	// Mw2: cmd="", options=[-g g0] start
	// GlobalHandler cmd="", options=[-g g0], -g=g0
	// CHandler cmd="c", options=[]
	// Mw2: cmd="", options=[-g g0] end
	// Mw2: cmd="", options=[] start
	// Not Found, args: cmd="b", options=[-no]
	// Mw2: cmd="", options=[] end
}

func TestApp(t *testing.T) {
	app := flagx.NewApp()
	app.SetName("TestApp")
	app.SetDescription("this is a app for testing")
	app.SetAuthors([]flagx.Author{{
		Name:  "henrylee2cn",
		Email: "henrylee2cn@gmail.com",
	}})
	app.Use(Mw1)
	app.Use(Mw2)

	app.SetOptions(new(GlobalHandler))
	app.MustAddAction("b", "test-b", new(BHandler))
	app.MustAddAction("a", "test-a", new(AHandler))
	app.MustAddAction("c", "test-c", flagx.HandlerFunc(CHandler))

	stat := app.Exec(context.TODO(), []string{"-h"})
	assert.NoError(t, stat.Cause())
	fmt.Printf("%+v\n\n", stat)

	stat = app.Exec(context.TODO(), []string{"a", "-a", "x"})
	assert.Empty(t, stat.Code())
	fmt.Printf("%+v\n\n", stat)

	stat = app.Exec(context.TODO(), []string{"b", "-b", "y"})
	assert.Empty(t, stat.Code())
	fmt.Printf("%+v\n\n", stat)

	stat = app.Exec(context.TODO(), []string{"c"})
	assert.Empty(t, stat.Code())
	fmt.Printf("%+v\n\n", stat)

	stat = app.Exec(context.TODO(), []string{"-g", "z", "--", "c"})
	assert.Empty(t, stat.Code())
	fmt.Printf("%+v\n\n", stat)

	app.SetNotFound(func(*flagx.Context) {
		fmt.Println("404:", app.UsageText())
	})
	stat = app.Exec(context.TODO(), []string{"x"})
	assert.Empty(t, stat.Code())
	fmt.Printf("%+v\n\n", stat)
}

func Mw1(c *flagx.Context, next flagx.HandlerFunc) {
	t := time.Now()
	argsInfo := c.ArgsInfo()
	fmt.Printf(
		"Mw1: cmd=%q, options=%v, start at=%v\n",
		argsInfo.Command, argsInfo.Options, t,
	)
	defer func() {
		fmt.Printf(
			"Mw1: cmd=%q, options=%v, cost time=%v\n",
			argsInfo.Command, argsInfo.Options, time.Since(t),
		)
	}()
	next(c)
}

func Mw2(c *flagx.Context, next flagx.HandlerFunc) {
	argsInfo := c.ArgsInfo()
	fmt.Printf(
		"Mw2: cmd=%q, options=%v start\n",
		argsInfo.Command, argsInfo.Options,
	)
	defer func() {
		fmt.Printf(
			"Mw2: cmd=%q, options=%v end\n",
			argsInfo.Command, argsInfo.Options,
		)
	}()
	next(c)
}

type GlobalHandler struct {
	G string `flag:"g;usage=GlobalHandler"`
}

func (g *GlobalHandler) Handle(c *flagx.Context) {
	argsInfo := c.ArgsInfo()
	fmt.Printf("GlobalHandler cmd=%q, options=%v, -g=%s\n", argsInfo.Command, argsInfo.Options, g.G)
}

type AHandler struct {
	A string `flag:"a;usage=AHandler"`
}

func (a *AHandler) Handle(c *flagx.Context) {
	argsInfo := c.ArgsInfo()
	fmt.Printf("AHandler cmd=%q, options=%v, -a=%s\n", argsInfo.Command, argsInfo.Options, a.A)
}

type BHandler struct {
	B string `flag:"b;usage=BHandler"`
}

func (b *BHandler) Handle(c *flagx.Context) {
	argsInfo := c.ArgsInfo()
	fmt.Printf("BHandler cmd=%q, options=%v, -b=%s\n", argsInfo.Command, argsInfo.Options, b.B)
}

func CHandler(c *flagx.Context) {
	argsInfo := c.ArgsInfo()
	fmt.Printf("CHandler cmd=%q, options=%v\n", argsInfo.Command, argsInfo.Options)
}
