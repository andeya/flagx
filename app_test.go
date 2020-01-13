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
		fmt.Printf("Not Found, args: %v", c.Args())
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

	stat = app.Exec(context.TODO(), []string{"b"})
	if !stat.OK() {
		panic(stat)
	}

	// Output:
	// Mw2: start map[a:[-a x]]
	// AHandler args=map[a:[-a x]], -a=x
	// Mw2: end map[a:[-a x]]
	// Mw2: start map[c:[]]
	// CHandler args:map[c:[]]
	// Mw2: end map[c:[]]
	// Mw2: start map[:[-g g0] c:[]]
	// GlobalHandler args=map[:[-g g0] c:[]], -g=g0
	// CHandler args:map[:[-g g0] c:[]]
	// Mw2: end map[:[-g g0] c:[]]
	// Not Found, args: map[b:[]]
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

func Mw1(c *flagx.Context, next func(*flagx.Context)) error {
	t := time.Now()
	fmt.Printf("Mw1: %+v, start at:%v\n", c.Args(), t)
	defer func() {
		fmt.Printf("Mw1: %+v, cost time:%s\n", c.Args(), time.Since(t))
	}()
	next(c)
	return nil
}

func Mw2(c *flagx.Context, next func(*flagx.Context)) error {
	fmt.Printf("Mw2: start %v\n", c.Args())
	defer func() {
		fmt.Printf("Mw2: end %v\n", c.Args())
	}()
	next(c)
	return nil
}

type GlobalHandler struct {
	G string `flag:"g;usage=GlobalHandler"`
}

func (g *GlobalHandler) Handle(c *flagx.Context) {
	fmt.Printf("GlobalHandler args=%+v, -g=%s\n", c.Args(), g.G)
}

type AHandler struct {
	A string `flag:"a;usage=AHandler"`
}

func (a *AHandler) Handle(c *flagx.Context) {
	fmt.Printf("AHandler args=%+v, -a=%s\n", c.Args(), a.A)
}

type BHandler struct {
	B string `flag:"b;usage=BHandler"`
}

func (b *BHandler) Handle(c *flagx.Context) {
	fmt.Printf("BHandler args=%+v, -b=%s\n", c.Args(), b.B)
}

func CHandler(c *flagx.Context) {
	fmt.Printf("CHandler args:%+v\n", c.Args())
}
