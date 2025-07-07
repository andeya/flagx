package flagx

import (
	"flag"
	"fmt"
	"os"
	"strings"
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
		arguments, _, err = filterArgs(arguments, func(key string, valPtr *string) bool {
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
	filteredArgs, _, _, _ := tidyArgs(arguments, func(key string) (want, next bool) {
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

// NextArgs returns arguments of the next subcommand.
func NextArgs() []string { return CommandLine.NextArgs() }

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

// NonBoolVar defines a bool non-flag with specified index, default value, and usage string.
// The argument p points to a bool variable in which to store the value of the non-flag.
func NonBoolVar(p *bool, index int, value bool, usage string) {
	CommandLine.NonVar(newBoolValue(value, p), index, usage)
}

// NonBool defines a bool non-flag with specified index, default value, and usage string.
// The return value is the address of a bool variable that stores the value of the non-flag.
func NonBool(index int, value bool, usage string) *bool {
	return CommandLine.NonBool(index, value, usage)
}

// NonIntVar defines an int non-flag with specified index, default value, and usage string.
// The argument p points to an int variable in which to store the value of the non-flag.
func NonIntVar(p *int, index int, value int, usage string) {
	CommandLine.NonVar(newIntValue(value, p), index, usage)
}

// NonInt defines an int non-flag with specified index, default value, and usage string.
// The return value is the address of an int variable that stores the value of the non-flag.
func NonInt(index int, value int, usage string) *int {
	return CommandLine.NonInt(index, value, usage)
}

// NonInt64Var defines an int64 non-flag with specified index, default value, and usage string.
// The argument p points to an int64 variable in which to store the value of the non-flag.
func NonInt64Var(p *int64, index int, value int64, usage string) {
	CommandLine.NonVar(newInt64Value(value, p), index, usage)
}

// NonInt64 defines an int64 non-flag with specified index, default value, and usage string.
// The return value is the address of an int64 variable that stores the value of the non-flag.
func NonInt64(index int, value int64, usage string) *int64 {
	return CommandLine.NonInt64(index, value, usage)
}

// NonUintVar defines a uint non-flag with specified index, default value, and usage string.
// The argument p points to a uint variable in which to store the value of the non-flag.
func NonUintVar(p *uint, index int, value uint, usage string) {
	CommandLine.NonVar(newUintValue(value, p), index, usage)
}

// NonUint defines a uint non-flag with specified index, default value, and usage string.
// The return value is the address of a uint variable that stores the value of the non-flag.
func NonUint(index int, value uint, usage string) *uint {
	return CommandLine.NonUint(index, value, usage)
}

// NonUint64Var defines a uint64 non-flag with specified index, default value, and usage string.
// The argument p points to a uint64 variable in which to store the value of the non-flag.
func NonUint64Var(p *uint64, index int, value uint64, usage string) {
	CommandLine.NonVar(newUint64Value(value, p), index, usage)
}

// NonUint64 defines a uint64 non-flag with specified index, default value, and usage string.
// The return value is the address of a uint64 variable that stores the value of the non-flag.
func NonUint64(index int, value uint64, usage string) *uint64 {
	return CommandLine.NonUint64(index, value, usage)
}

// NonStringVar defines a string non-flag with specified index, default value, and usage string.
// The argument p points to a string variable in which to store the value of the non-flag.
func NonStringVar(p *string, index int, value string, usage string) {
	CommandLine.NonVar(newStringValue(value, p), index, usage)
}

// NonString defines a string non-flag with specified index, default value, and usage string.
// The return value is the address of a string variable that stores the value of the non-flag.
func NonString(index int, value string, usage string) *string {
	return CommandLine.NonString(index, value, usage)
}

// NonFloat64Var defines a float64 non-flag with specified index, default value, and usage string.
// The argument p points to a float64 variable in which to store the value of the non-flag.
func NonFloat64Var(p *float64, index int, value float64, usage string) {
	CommandLine.NonVar(newFloat64Value(value, p), index, usage)
}

// NonFloat64 defines a float64 non-flag with specified index, default value, and usage string.
// The return value is the address of a float64 variable that stores the value of the non-flag.
func NonFloat64(index int, value float64, usage string) *float64 {
	return CommandLine.NonFloat64(index, value, usage)
}

// NonDurationVar defines a time.Duration non-flag with specified index, default value, and usage string.
// The argument p points to a time.Duration variable in which to store the value of the non-flag.
// The non-flag accepts a value acceptable to time.ParseDuration.
func NonDurationVar(p *time.Duration, index int, value time.Duration, usage string) {
	CommandLine.NonVar(newDurationValue(value, p), index, usage)
}

// NonDuration defines a time.Duration non with specified index, default value, and usage string.
// The return value is the address of a time.Duration variable that stores the value of the non-flag.
// The non-flag accepts a value acceptable to time.ParseDuration.
func NonDuration(index int, value time.Duration, usage string) *time.Duration {
	return CommandLine.NonDuration(index, value, usage)
}

// NonVar defines a non-flag with the specified index and usage string.
func NonVar(value Value, index int, usage string) {
	CommandLine.NonVar(value, index, usage)
}

// NArg is the number of arguments remaining after flags have been processed.
func NArg() int {
	return CommandLine.NArg()
}

// NFlag returns the number of command-line flags that have been set.
func NFlag() int {
	return CommandLine.NFlag()
}

// NFormalNonFlag returns the number of non-flag required in the definition.
func NFormalNonFlag() int {
	return CommandLine.NFormalNonFlag()
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
//
//	-x int
//		usage-message-for-x (default 7)
//
// The usage message will appear on a separate line for anything but
// a bool flag with a one-byte name. For bool flags, the type is
// omitted and if the flag name is one byte the usage message appears
// on the same line. The parenthetical default is omitted if the
// default is the zero value for the type. The listed type, here int,
// can be changed by placing a back-quoted name in the flag's usage
// string; the first such item in the message is taken to be a parameter
// name to show in the message and the back quotes are stripped from
// the message when displayed. For instance, given
//
//	flag.String("I", "", "search `directory` for include files")
//
// the output will be
//
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
//
//	Not support nested fields
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
	if !IsNonFlag(f) {
		return flag.UnquoteUsage(f)
	}
	// Look for a back-quoted name, but avoid the strings package.
	usage = f.Usage
	for i := 0; i < len(usage); i++ {
		if usage[i] == '`' {
			for j := i + 1; j < len(usage); j++ {
				if usage[j] == '`' {
					name = usage[i+1 : j]
					usage = usage[:i] + name + usage[j+1:]
					return name, usage
				}
			}
			break // Only one back quote; use type name.
		}
	}
	// No explicit name, so use type if we can find one.
	name = "value"
	switch f.Value.(type) {
	case boolFlag:
		name = "bool"
	case *durationValue:
		name = "duration"
	case *float64Value:
		name = "float"
	case *intValue, *int64Value:
		name = "int"
	case *stringValue:
		name = "string"
	case *uintValue, *uint64Value:
		name = "uint"
	}
	return
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

// RangeAll visits the command-line flags and non-flags in lexicographical order, calling fn for each.
// It visits all flags and non-flags, even those not set.
func RangeAll(fn func(*Flag)) {
	CommandLine.RangeAll(fn)
}

// Range visits the command-line flags and non-flags in lexicographical order, calling fn for each.
// It visits only those flags and non-flags that have been set.
func Range(fn func(*Flag)) {
	CommandLine.Range(fn)
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

// NonVisitAll visits the command-line non-flags in lexicographical order, calling
// fn for each. It visits all non-flags, even those not set.
func NonVisitAll(fn func(*Flag)) {
	CommandLine.NonVisitAll(fn)
}

// NonVisit visits the command-line non-flags in lexicographical order, calling fn
// for each. It visits only those non-flags that have been set.
func NonVisit(fn func(*Flag)) {
	CommandLine.NonVisit(fn)
}

// IsNonFlag determines if it is non-flag.
func IsNonFlag(f *Flag) bool {
	return strings.HasPrefix(f.Name, "?")
}

// NonFlagIndex gets the non-flag index from name.
func NonFlagIndex(nonFlag *Flag) (int, bool) {
	idx, _, _ := getNonFlagIndex(nonFlag.Name)
	return idx, idx >= 0
}
