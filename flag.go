package flagx

import (
	"flag"
	"fmt"
	"reflect"
	"strings"

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

// StructVars defines flags based on struct tags and binds to fields.
// NOTE:
//  Not support nested fields
func (f *FlagSet) StructVars(p interface{}) error {
	v := reflect.ValueOf(p)
	if v.Kind() == reflect.Ptr {
		v = goutil.DereferenceValue(v)
		if v.Kind() == reflect.Struct {
			return f.varFromStruct(v)
		}
	}
	return fmt.Errorf("flagx: want struct pointer parameter, but got %T", p)
}

// Parse parses flag definitions from the argument list, which should not
// include the command name. Must be called after all flags in the FlagSet
// are defined and before flags are accessed by the program.
// The return value will be ErrHelp if -help or -h were set but not defined.
func (f *FlagSet) Parse(arguments []string) error {
	if f.isContinueOnUndefined {
		names := make([]string, 0, len(arguments))
		f.FlagSet.VisitAll(func(f *Flag) {
			names = append(names, f.Name)
		})
		arguments = filterArgs(arguments, names)
	}
	return f.FlagSet.Parse(arguments)
}

func filterArgs(args, names []string) []string {
	a := make([]string, 0, len(names))
L:
	for _, name := range names {
		for i := len(args) - 1; i >= 0; i-- {
			if !strings.HasPrefix(args[i], "-") {
				continue
			}
			key := strings.TrimLeft(args[i], "-")
			idx := strings.Index(key, "=")
			if idx == -1 {
				if key != name {
					continue
				}
				if i+1 < len(args) {
					val := args[i+1]
					if !strings.HasPrefix(val, "-") {
						a = append(a, "-"+name, val)
						continue L
					}
				}
				a = append(a, "-"+name)
				continue L
			}
			if key[:idx] != name {
				continue
			}
			a = append(a, "-"+name, key[idx+1:])
			continue L
		}
	}
	return a
}

func cleanBit(eh, bit ErrorHandling) (ErrorHandling, bool) {
	eh2 := eh &^ bit
	return eh2, eh2 != eh
}
