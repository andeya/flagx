# flagx [![report card](https://goreportcard.com/badge/github.com/henrylee2cn/flagx?style=flat-square)](http://goreportcard.com/report/henrylee2cn/flagx) [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/henrylee2cn/flagx)

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
		cmdName, options := c.Args()
		fmt.Printf(
			"Not Found, args: cmd=%q, options=%v\n",
			cmdName, options,
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

func Mw2(c *flagx.Context, next flagx.HandlerFunc) {
	cmdName, options := c.Args()
	fmt.Printf(
		"Mw2: cmd=%q, options=%v start\n",
		cmdName, options,
	)
	defer func() {
		fmt.Printf(
			"Mw2: cmd=%q, options=%v end\n",
			cmdName, options,
		)
	}()
	next(c)
}

type GlobalHandler struct {
	G string `flag:"g;usage=GlobalHandler"`
}

func (g *GlobalHandler) Handle(c *flagx.Context) {
	cmdName, options := c.Args()
	fmt.Printf("GlobalHandler cmd=%q, options=%v, -g=%s\n", cmdName, options, g.G)
}

type AHandler struct {
	A string `flag:"a;usage=AHandler"`
}

func (a *AHandler) Handle(c *flagx.Context) {
	cmdName, options := c.Args()
	fmt.Printf("AHandler cmd=%q, options=%v, -a=%s\n", cmdName, options, a.A)
}

func CHandler(c *flagx.Context) {
	cmdName, options := c.Args()
	fmt.Printf("CHandler cmd=%q, options=%v\n", cmdName, options)
}
```