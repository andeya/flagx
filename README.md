# flagx [![report card](https://goreportcard.com/badge/github.com/henrylee2cn/flagx?style=flat-square)](http://goreportcard.com/report/henrylee2cn/flagx) [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/henrylee2cn/flagx)

Standard flag package extension with more free usage.

## Extension Feature

- Add `const ContinueOnUndefined ErrorHandling` to ignore provided but undefined flags
- Add `*FlagSet.StructVars` to define flags based on struct tags and bind to fields
- For more features, please open the issue

## Test Demo

- ignore provided but undefined flags

```go
func TestContinueOnUndefined(t *testing.T) {
	fs := flagx.NewFlagSet(os.Args[0], ContinueOnError)
	run := fs.String("test.run", "", "")
	err := fs.Parse(os.Args[1:])
	assert.NotNil(t, err)
	t.Log(err)

	fs = flagx.NewFlagSet(os.Args[0], ContinueOnError|ContinueOnUndefined)
	run = fs.String("test.run", "", "")
	err = fs.Parse(os.Args[1:])
	assert.NoError(t, err)
	assert.True(t, strings.Contains(*run, "TestContinueOnUndefined"))
}
```

- define flags based on struct tags and bind to fields

```go
func TestStructVars(t *testing.T) {
	fs := flagx.NewFlagSet(os.Args[0], ContinueOnError|ContinueOnUndefined)
	type Args struct {
		Run     string        `flag:"test.run; def=.*; usage=function name pattern"`
		Timeout time.Duration `flag:"test.timeout"`
	}
	var args Args
	err := fs.StructVars(&args)
	assert.NoError(t, err)
	err = fs.Parse(os.Args[1:])
	assert.NoError(t, err)
	assert.True(t, strings.Contains(args.Run, "TestStructVars"))
	t.Logf("%+v", args)
	fs.Usage()
}
```
