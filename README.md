# flagx [![report card](https://goreportcard.com/badge/github.com/henrylee2cn/flagx?style=flat-square)](http://goreportcard.com/report/henrylee2cn/flagx) [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/henrylee2cn/flagx)

Standard flag package extension with more free usage.

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
- For more features, please open the issue

## Test Demo

- ignore provided but undefined flags

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

- define flags based on struct tags and bind to fields

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

- lookup the value corresponding to a name directly from arguments

```go
func TestLookupArgs(t *testing.T) {
	var args = []string{"-run", "abc", "-t", "5s", "-Cool", "-N", "1"}

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

	v, ok = LookupArgs(args, "???")
	assert.False(t, ok)
	assert.Equal(t, "", v)
}
```