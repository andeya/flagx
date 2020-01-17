package flagx

import (
	"flag"
	"fmt"
	"os"
	"time"
)

// SplitArgs returns the command name and options.
func SplitArgs(arguments []string) (string, []string) {
	if len(arguments) > 0 {
		if s := arguments[0]; len(s) > 0 && s[0] != '-' {
			return s, arguments[1:]
		}
	}
	return "", arguments
}

// Option command option
type Option struct {
	Command string
	Name    string
	Value   string
}

// LookupOptions lookups the options corresponding to the name
// directly from the arguments.
func LookupOptions(arguments []string, name string) []*Option {
	if name == "" {
		return nil
	}
	r := make([]*Option, 0, 2)
	var err error
	var cmd string
	for {
		cmd, arguments = SplitArgs(arguments)
		arguments, err = filterArgs(arguments, func(key string, valPtr *string) bool {
			if key == name {
				var val string
				if valPtr != nil {
					val = *valPtr
				}
				r = append(r, &Option{
					Command: cmd,
					Name:    name,
					Value:   val,
				})
			}
			return true
		})
		if err != nil || len(arguments) == 0 {
			return r
		}
	}
}

// LookupArgs lookups the value corresponding to the name
// directly from the arguments.
func LookupArgs(arguments []string, name string) (value string, found bool) {
	_, arguments = SplitArgs(arguments)
	filteredArgs, _, _ := tidyArgs(arguments, func(key string) (want, next bool) {
		if key == name {
			return true, false
		}
		return false, true
	})
	switch len(filteredArgs) {
	case 0:
		return "", false
	case 1:
		return "", true
	default:
		return filteredArgs[1], true
	}
}

// Lookup returns the Flag structure of the named command-line flag,
// returning nil if none exists.
func Lookup(name string) *Flag {
	return CommandLine.Lookup(name)
}

// CommandLine is the default set of command-line flags, parsed from os.Args.
// The top-level functions such as BoolVar, Arg, and so on are wrappers for the
// methods of CommandLine.
var CommandLine = NewFlagSet(os.Args[0], ExitOnError|ContinueOnUndefined)

func init() {
	// Override generic FlagSet default Usage with call to global Usage.
	// Note: This is not CommandLine.Usage = Usage,
	// because we want any eventual call to use any updated value of Usage,
	// not the value it has when this line is run.
	CommandLine.Usage = flag.CommandLine.Usage
}

// Arg returns the i'th command-line argument. Arg(0) is the first remaining argument
// after flags have been processed. Arg returns an empty string if the
// requested element does not exist.
func Arg(i int) string {
	return CommandLine.Arg(i)
}

// Args returns the non-flag command-line arguments.
func Args() []string {
	return CommandLine.Args()
}

// Bool defines a bool flag with specified name, default value, and usage string.
// The return value is the address of a bool variable that stores the value of the flag.
func Bool(name string, value bool, usage string) *bool {
	return CommandLine.Bool(name, value, usage)
}

// BoolVar defines a bool flag with specified name, default value, and usage string.
// The argument p points to a bool variable in which to store the value of the flag.
func BoolVar(p *bool, name string, value bool, usage string) {
	CommandLine.BoolVar(p, name, value, usage)
}

// Duration defines a time.Duration flag with specified name, default value, and usage string.
// The return value is the address of a time.Duration variable that stores the value of the flag.
// The flag accepts a value acceptable to time.ParseDuration.
func Duration(name string, value time.Duration, usage string) *time.Duration {
	return CommandLine.Duration(name, value, usage)
}

// DurationVar defines a time.Duration flag with specified name, default value, and usage string.
// The argument p points to a time.Duration variable in which to store the value of the flag.
// The flag accepts a value acceptable to time.ParseDuration.
func DurationVar(p *time.Duration, name string, value time.Duration, usage string) {
	CommandLine.DurationVar(p, name, value, usage)
}

// Float64 defines a float64 flag with specified name, default value, and usage string.
// The return value is the address of a float64 variable that stores the value of the flag.
func Float64(name string, value float64, usage string) *float64 {
	return CommandLine.Float64(name, value, usage)
}

// Float64Var defines a float64 flag with specified name, default value, and usage string.
// The argument p points to a float64 variable in which to store the value of the flag.
func Float64Var(p *float64, name string, value float64, usage string) {
	CommandLine.Float64Var(p, name, value, usage)
}

// Int defines an int flag with specified name, default value, and usage string.
// The return value is the address of an int variable that stores the value of the flag.
func Int(name string, value int, usage string) *int {
	return CommandLine.Int(name, value, usage)
}

// Int64 defines an int64 flag with specified name, default value, and usage string.
// The return value is the address of an int64 variable that stores the value of the flag.
func Int64(name string, value int64, usage string) *int64 {
	return CommandLine.Int64(name, value, usage)
}

// Int64Var defines an int64 flag with specified name, default value, and usage string.
// The argument p points to an int64 variable in which to store the value of the flag.
func Int64Var(p *int64, name string, value int64, usage string) {
	CommandLine.Int64Var(p, name, value, usage)
}

// IntVar defines an int flag with specified name, default value, and usage string.
// The argument p points to an int variable in which to store the value of the flag.
func IntVar(p *int, name string, value int, usage string) {
	CommandLine.IntVar(p, name, value, usage)
}

// NArg is the number of arguments remaining after flags have been processed.
func NArg() int {
	return CommandLine.NArg()
}

// NFlag returns the number of command-line flags that have been set.
func NFlag() int {
	return CommandLine.NFlag()
}

// Parse parses the command-line flags from os.Args[1:]. Must be called
// after all flags are defined and before flags are accessed by the program.
func Parse() {
	// Ignore errors; CommandLine is set for ExitOnError.
	CommandLine.Parse(os.Args[1:])
}

// Parsed reports whether the command-line flags have been parsed.
func Parsed() bool {
	return CommandLine.Parsed()
}

// Usage prints the default usage message.
func Usage() {
	if CommandLine.Usage != nil {
		CommandLine.Usage()
	} else {
		if CommandLine.Name() == "" {
			fmt.Fprintf(CommandLine.Output(), "Usage:\n")
		} else {
			fmt.Fprintf(CommandLine.Output(), "Usage of %s:\n", CommandLine.Name())
		}
		CommandLine.PrintDefaults()
	}
}

// PrintDefaults prints, to standard error unless configured otherwise,
// a usage message showing the default settings of all defined
// command-line flags.
// For an integer valued flag x, the default output has the form
//	-x int
//		usage-message-for-x (default 7)
// The usage message will appear on a separate line for anything but
// a bool flag with a one-byte name. For bool flags, the type is
// omitted and if the flag name is one byte the usage message appears
// on the same line. The parenthetical default is omitted if the
// default is the zero value for the type. The listed type, here int,
// can be changed by placing a back-quoted name in the flag's usage
// string; the first such item in the message is taken to be a parameter
// name to show in the message and the back quotes are stripped from
// the message when displayed. For instance, given
//	flag.String("I", "", "search `directory` for include files")
// the output will be
//	-I directory
//		search directory for include files.
//
// To change the destination for flag messages, call CommandLine.SetOutput.
func PrintDefaults() {
	CommandLine.PrintDefaults()
}

// Set sets the value of the named command-line flag.
func Set(name, value string) error {
	return CommandLine.Set(name, value)
}

// String defines a string flag with specified name, default value, and usage string.
// The return value is the address of a string variable that stores the value of the flag.
func String(name string, value string, usage string) *string {
	return CommandLine.String(name, value, usage)
}

// StringVar defines a string flag with specified name, default value, and usage string.
// The argument p points to a string variable in which to store the value of the flag.
func StringVar(p *string, name string, value string, usage string) {
	CommandLine.StringVar(p, name, value, usage)
}

// StructVars defines flags based on struct tags and binds to fields.
// NOTE:
//  Not support nested fields
func StructVars(p interface{}) error {
	return CommandLine.StructVars(p)
}

// Uint defines a uint flag with specified name, default value, and usage string.
// The return value is the address of a uint variable that stores the value of the flag.
func Uint(name string, value uint, usage string) *uint {
	return CommandLine.Uint(name, value, usage)
}

// Uint64 defines a uint64 flag with specified name, default value, and usage string.
// The return value is the address of a uint64 variable that stores the value of the flag.
func Uint64(name string, value uint64, usage string) *uint64 {
	return CommandLine.Uint64(name, value, usage)
}

// Uint64Var defines a uint64 flag with specified name, default value, and usage string.
// The argument p points to a uint64 variable in which to store the value of the flag.
func Uint64Var(p *uint64, name string, value uint64, usage string) {
	CommandLine.Uint64Var(p, name, value, usage)
}

// UintVar defines a uint flag with specified name, default value, and usage string.
// The argument p points to a uint variable in which to store the value of the flag.
func UintVar(p *uint, name string, value uint, usage string) {
	CommandLine.UintVar(p, name, value, usage)
}

// UnquoteUsage extracts a back-quoted name from the usage
// string for a flag and returns it and the un-quoted usage.
// Given "a `name` to show" it returns ("name", "a name to show").
// If there are no back quotes, the name is an educated guess of the
// type of the flag's value, or the empty string if the flag is boolean.
func UnquoteUsage(f *Flag) (name string, usage string) {
	return flag.UnquoteUsage(f)
}

// Var defines a flag with the specified name and usage string. The type and
// value of the flag are represented by the first argument, of type Value, which
// typically holds a user-defined implementation of Value. For instance, the
// caller could create a flag that turns a comma-separated string into a slice
// of strings by giving the slice the methods of Value; in particular, Set would
// decompose the comma-separated string into the slice.
func Var(value Value, name string, usage string) {
	CommandLine.Var(value, name, usage)
}

// Visit visits the command-line flags in lexicographical order, calling fn
// for each. It visits only those flags that have been set.
func Visit(fn func(*Flag)) {
	CommandLine.Visit(fn)
}

// VisitAll visits the command-line flags in lexicographical order, calling
// fn for each. It visits all flags, even those not set.
func VisitAll(fn func(*Flag)) {
	CommandLine.VisitAll(fn)
}
