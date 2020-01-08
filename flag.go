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

// struct tags are used by *FlagSet.StructVars.
const (
	tagNameFlag       = "flag"
	tagKeyOmit        = "-"
	tagKeyNameDefault = "def"
	tagKeyNameUsage   = "usage"
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
		f.FlagSet.Usage = f.defaultUsage
	} else {
		f.FlagSet.Init(name, errorHandling)
		if f.FlagSet.Usage == nil {
			f.FlagSet.Usage = f.defaultUsage
		}
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

// PrintDefaults prints, to standard error unless configured otherwise, the
// default values of all defined command-line flags in the set. See the
// documentation for the global function PrintDefaults for more information.
func (f *FlagSet) PrintDefaults() {
	f.VisitAll(func(flag *Flag) {
		s := fmt.Sprintf("  -%s", flag.Name) // Two spaces before -; see next two comments.
		name, usage := UnquoteUsage(flag)
		if len(name) > 0 {
			s += " " + name
		}
		// Boolean flags of one ASCII letter are so common we
		// treat them specially, putting their usage on the same line.
		if len(s) <= 4 { // space, space, '-', 'x'.
			s += "\t"
		} else {
			// Four spaces before the tab triggers good alignment
			// for both 4- and 8-space tab stops.
			s += "\n    \t"
		}
		s += strings.ReplaceAll(usage, "\n", "\n    \t")

		if !isZeroValue(flag, flag.DefValue) {
			if _, ok := flag.Value.(*stringValue); ok {
				// put quotes on the value
				s += fmt.Sprintf(" (default %q)", flag.DefValue)
			} else {
				s += fmt.Sprintf(" (default %v)", flag.DefValue)
			}
		}
		fmt.Fprint(f.Output(), s, "\n")
	})
}

// defaultUsage is the default function to print a usage message.
func (f *FlagSet) defaultUsage() {
	name := f.FlagSet.Name()
	if name == "" {
		fmt.Fprintf(f.Output(), "Usage:\n")
	} else {
		fmt.Fprintf(f.Output(), "Usage of %s:\n", name)
	}
	f.PrintDefaults()
}

// isZeroValue determines whether the string represents the zero
// value for a flag.
func isZeroValue(flag *Flag, value string) bool {
	// Build a zero value of the flag's Value type, and see if the
	// result of calling its String method equals the value passed in.
	// This works unless the Value type is itself an interface type.
	typ := reflect.TypeOf(flag.Value)
	var z reflect.Value
	if typ.Kind() == reflect.Ptr {
		z = reflect.New(typ.Elem())
	} else {
		z = reflect.Zero(typ)
	}
	return value == z.Interface().(Value).String()
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
				a = append(a, "-"+name, "")
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
