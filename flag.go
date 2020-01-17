package flagx

import (
	"flag"
	"fmt"
	"reflect"

	"github.com/henrylee2cn/goutil"
)

type (
	// ErrorHandling defines how FlagSet.Parse behaves if the parse fails.
	ErrorHandling = flag.ErrorHandling

	// A FlagSet represents a set of defined flags. The zero value of a FlagSet
	// has no name and has ContinueOnError error handling.
	FlagSet struct {
		*flag.FlagSet
		errorHandling         ErrorHandling
		isContinueOnUndefined bool
	}

	// A Flag represents the state of a flag.
	Flag = flag.Flag

	// Getter is an interface that allows the contents of a Value to be retrieved.
	// It wraps the Value interface, rather than being part of it, because it
	// appeared after Go 1 and its compatibility rules. All Value types provided
	// by this package satisfy the Getter interface.
	Getter = flag.Getter

	// Value is the interface to the dynamic value stored in a flag.
	// (The default value is represented as a string.)
	//
	// If a Value has an IsBoolFlag() bool method returning true,
	// the command-line parser makes -name equivalent to -name=true
	// rather than using the next command-line argument.
	//
	// Set is called once, in command line order, for each flag present.
	// The flag package may call the String method with a zero-valued receiver,
	// such as a nil pointer.
	Value = flag.Value
)

// These constants cause FlagSet.Parse to behave as described if the parse fails.
const (
	ContinueOnError     ErrorHandling = flag.ContinueOnError // Return a descriptive error.
	ExitOnError         ErrorHandling = flag.ExitOnError     // Call os.Exit(2).
	PanicOnError        ErrorHandling = flag.PanicOnError    // Call panic with a descriptive error.
	ContinueOnUndefined ErrorHandling = 1 << 30              // Ignore provided but undefined flags
)

// NewFlagSet returns a new, empty flag set with the specified name and
// error handling property. If the name is not empty, it will be printed
// in the default usage message and in error messages.
func NewFlagSet(name string, errorHandling ErrorHandling) *FlagSet {
	f := new(FlagSet)
	f.Init(name, errorHandling)
	return f
}

// Init sets the name and error handling property for a flag set.
// By default, the zero FlagSet uses an empty name and the
// ContinueOnError error handling policy.
func (f *FlagSet) Init(name string, errorHandling ErrorHandling) {
	f.errorHandling = errorHandling
	errorHandling, f.isContinueOnUndefined = cleanBit(errorHandling, ContinueOnUndefined)
	if f.FlagSet == nil {
		f.FlagSet = flag.NewFlagSet(name, errorHandling)
	} else {
		f.FlagSet.Init(name, errorHandling)
	}
}

// ErrorHandling returns the error handling behavior of the flag set.
func (f *FlagSet) ErrorHandling() ErrorHandling {
	return f.errorHandling
}

// StructVars defines flags based on struct tags and binds to fields.
// NOTE:
//  Not support nested fields
func (f *FlagSet) StructVars(p interface{}) error {
	v := reflect.ValueOf(p)
	if v.Kind() == reflect.Ptr {
		v = goutil.DereferenceValue(v)
		if v.Kind() == reflect.Struct {
			structTypeIDs := make(map[int32]struct{}, 4)
			return f.varFromStruct(v, structTypeIDs)
		}
	}
	return fmt.Errorf("flagx: want struct pointer parameter, but got %T", p)
}

// Parse parses flag definitions from the argument list, which should not
// include the command name. Must be called after all flags in the FlagSet
// are defined and before flags are accessed by the program.
// The return value will be ErrHelp if -help or -h were set but not defined.
func (f *FlagSet) Parse(arguments []string) error {
	_, arguments = SplitArgs(arguments)
	if f.isContinueOnUndefined {
		var err error
		arguments, _, err = tidyArgs(arguments, func(name string) (want, next bool) {
			return f.FlagSet.Lookup(name) != nil, true
		})
		if err != nil {
			return err
		}
	}
	return f.FlagSet.Parse(arguments)
}

func tidyArgs(args []string, filter func(name string) (want, next bool)) (tidiedArgs, lastArgs []string, err error) {
	tidiedArgs = make([]string, 0, len(args)*2)
	lastArgs, err = filterArgs(args, func(name string, valuePtr *string) bool {
		want, next := filter(name)
		if want {
			var kv []string
			if valuePtr == nil {
				kv = []string{"-" + name}
			} else {
				kv = []string{"-" + name, *valuePtr}
			}
			tidiedArgs = append(tidiedArgs, kv...)
		}
		return next
	})
	return tidiedArgs, lastArgs, err
}

func filterArgs(args []string, filter func(name string, valuePtr *string) (next bool)) (lastArgs []string, err error) {
	lastArgs = args
	var name string
	var valuePtr *string
	var seen bool
	for {
		lastArgs, name, valuePtr, seen, err = tidyOneArg(lastArgs)
		if !seen {
			return
		}
		next := filter(name, valuePtr)
		if !next {
			return
		}
	}
}

// tidyOneArg tidies one flag. It reports whether a flag was seen.
func tidyOneArg(args []string) (lastArgs []string, name string, valuePtr *string, seen bool, err error) {
	if len(args) == 0 {
		lastArgs = args
		return
	}
	s := args[0]
	if len(s) < 2 || s[0] != '-' {
		lastArgs = args
		return
	}
	numMinuses := 1
	if s[1] == '-' {
		numMinuses++
		if len(s) == 2 { // "--" terminates the flags
			lastArgs = args[1:]
			return
		}
	}
	name = s[numMinuses:]
	if len(name) == 0 || name[0] == '-' || name[0] == '=' {
		err = fmt.Errorf("bad flag syntax: %s", s)
		lastArgs = args
		return
	}

	// it's a flag.
	seen = true
	args = args[1:]

	// does it have an argument?
	for i := 1; i < len(name); i++ { // equals cannot be first
		if name[i] == '=' {
			value := name[i+1:]
			valuePtr = &value
			name = name[0:i]
			lastArgs = args
			return
		}
	}

	// doesn't have an arg
	if len(args) == 0 {
		lastArgs = args
		return
	}

	// value is the next arg
	if maybeValue := args[0]; len(maybeValue) == 0 || maybeValue[0] != '-' {
		valuePtr = &maybeValue
		lastArgs = args[1:]
		return
	}

	// doesn't have an arg
	lastArgs = args
	return
}

func cleanBit(eh, bit ErrorHandling) (ErrorHandling, bool) {
	eh2 := eh &^ bit
	return eh2, eh2 != eh
}
