package flagx

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestApp(t *testing.T) {
	app := &App{}

	app.Use(Mw1)
	app.Use(Mw2)

	app.MustReg("", "test-global", new(GlobalHandler))
	app.MustReg("b", "test-b", new(BHandler))
	app.MustReg("a", "test-a", new(AHandler))

	fmt.Println(app.Usage())

	stat := app.Exec(context.TODO(), []string{"a", "-a", "x"})
	assert.NoError(t, stat.Cause())
	fmt.Printf("%+v\n\n", stat)

	stat = app.Exec(context.TODO(), []string{"b", "-b", "y"})
	assert.NoError(t, stat.Cause())
	fmt.Printf("%+v\n\n", stat)

	stat = app.Exec(context.TODO(), []string{"-g", "z", "--", "b", "-b", "y"})
	assert.NoError(t, stat.Cause())
	fmt.Printf("%+v\n\n", stat)
}

func Mw1(c *Context, next func(*Context)) error {
	t := time.Now()
	fmt.Printf("Mw1: %+v, start at:%v\n", c.Args(), t)
	defer func() {
		fmt.Printf("Mw1: %+v, cost time:%s\n", c.Args(), time.Since(t))
	}()
	next(c)
	return nil
}

func Mw2(c *Context, next func(*Context)) error {
	t := time.Now()
	fmt.Printf("Mw2: %+v, start at:%v\n", c.Args(), t)
	defer func() {
		fmt.Printf("Mw2: %+v, cost time:%s\n", c.Args(), time.Since(t))
	}()
	next(c)
	return nil
}

type GlobalHandler struct {
	G string `flag:"g;usage=GlobalHandler"`
}

func (*GlobalHandler) Handle(c *Context) {
	fmt.Printf("GlobalHandler args:%+v\n", c.Args())
}

type AHandler struct {
	A string `flag:"a;usage=AHandler"`
}

func (*AHandler) Handle(c *Context) {
	fmt.Printf("AHandler args:%+v\n", c.Args())
}

type BHandler struct {
	B string `flag:"b;usage=BHandler"`
}

func (*BHandler) Handle(c *Context) {
	fmt.Printf("BHandler args:%+v\n", c.Args())
}
