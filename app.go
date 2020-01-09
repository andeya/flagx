package flagx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/status"
)

type (
	// App is the main structure of a cli application. It is recommended that
	// an app be created with the cli.NewApp() function
	App struct {
		// The name of the program. Defaults to path.Base(os.Args[0])
		Name string
		// Full name of command for help, defaults to Name
		HelpName string
		// Version of the program
		Version string
		// Description of the program
		Description string
		// Compilation date
		Compiled time.Time
		// List of all authors who contributed
		Authors []Author
		// Copyright of the binary if any
		Copyright   string
		middlewares []Middleware
		// Execute this function if the proper command cannot be found
		notFoundHandler Handler
		actions         map[string]*Action
		usageTest       string
	}

	// Author represents someone who has contributed to a cli project.
	Author struct {
		Name  string // The Authors name
		Email string // The Authors email
	}
	Handler interface{ Handle(*Context) }
	// Middleware middleware of an action execution
	Middleware func(c *Context, next func(*Context)) error
	// Context context of an action execution
	Context struct {
		context.Context
		args []string
	}
	Action struct {
		flagSet               *FlagSet
		description           string
		usageBody             string
		usageText             string
		handlerStructElemType reflect.Type
	}
)

func (a *App) Use(mw Middleware) {
	a.middlewares = append(a.middlewares, mw)
}
func (a *App) MustReg(cmdName, desc string, handlerStruct Handler) {
	err := a.Reg(cmdName, desc, handlerStruct)
	if err != nil {
		panic(err)
	}
}
func (a *App) Reg(cmdName, desc string, handlerStruct Handler) error {
	action, err := newAction(cmdName, desc, handlerStruct)
	if err != nil {
		return err
	}
	if a.actions == nil {
		a.actions = make(map[string]*Action)
	}
	a.actions[cmdName] = action
	return nil
}

func (a *App) Exec(ctx context.Context, arguments []string) (stat *status.Status) {
	defer status.Catch(&stat)
	var c = &Context{args: arguments, Context: ctx}
	argsGroup, err := pickCommandAndArguments(arguments)
	status.Check(err, 1, "")
	var actions = make([]*Action, 0, 2)
	for _, g := range argsGroup {
		if len(g) == 0 {
			continue
		}
		subcommand := g[0]
		action := a.actions[subcommand]
		if action == nil {
			if subcommand == "" {
				status.Throw(1, "not support global flags", nil)
			}
			status.Throw(2, fmt.Sprintf("subcommand %q is not defined", subcommand), nil)
		}
		actions = append(actions, action)
	}
	handle := func(c *Context) {
		for _, action := range actions {
			action.exec(c)
		}
	}
	for i := len(a.middlewares) - 1; i >= 0; i-- {
		middleware := a.middlewares[i]
		nextHandle := handle
		handle = func(c *Context) {
			middleware(c, nextHandle)
		}
	}
	handle(c)
	return nil
}

func (a *App) Usage() string {
	if a.usageTest == "" {
		name := a.Name
		if name == "" {
			name = filepath.Base(os.Args[0])
		}
		var version = strings.TrimPrefix(a.Version, "v")
		if version == "" {
			version = "v0.0.1"
		}
		a.usageTest += fmt.Sprintf("%s %s\n", name, version)
		if len(a.actions) > 0 {
			a.usageTest += fmt.Sprintf("\nUsage: %s [-global_arguments --] [ subcommand ] [-sub_arguments]", name)
			nameList := make([]string, 0, len(a.actions))
			for name := range a.actions {
				nameList = append(nameList, name)
			}
			sort.Strings(nameList)
			if nameList[0] == "" {
				a.usageTest += "\n\n" + "Usage of global arguments:\n"
				a.usageTest += "\n" + a.actions[nameList[0]].UsageText()
				nameList = nameList[1:]
			}
			if len(nameList) > 0 {
				a.usageTest += "\n\n" + "Usage of subcommands:\n"
				for _, name := range nameList {
					a.usageTest += "\n" + a.actions[name].UsageText()
				}
			}
		}
		a.usageTest = strings.Replace(a.usageTest, "\n\n\n", "\n\n", -1)
	}
	return a.usageTest
}

func newAction(cmdName, desc string, handlerStruct Handler) (*Action, error) {
	var srv Action
	srv.flagSet = NewFlagSet(cmdName, ContinueOnError|ContinueOnUndefined)
	err := srv.flagSet.StructVars(handlerStruct)
	if err != nil {
		return nil, err
	}

	srv.description = desc
	srv.handlerStructElemType = goutil.DereferenceType(reflect.TypeOf(handlerStruct))

	var buf bytes.Buffer
	srv.flagSet.SetOutput(&buf)
	srv.flagSet.PrintDefaults()
	srv.usageBody = buf.String()
	srv.usageText += fmt.Sprintf("%s:\t%s\n", cmdName, desc)
	srv.usageText += srv.usageBody
	srv.flagSet.SetOutput(ioutil.Discard)
	return &srv, nil
}

func (a *Action) UsageText() string {
	return a.usageText
}

func (a *Action) CmdName() string {
	return a.flagSet.Name()
}

func (a *Action) Description() string {
	return a.description
}

func (a *Action) exec(c *Context) {
	handlerStruct := a.newHandlerStruct()
	flagSet := NewFlagSet(a.flagSet.Name(), a.flagSet.ErrorHandling())
	flagSet.StructVars(handlerStruct)
	handlerStruct.Handle(c)
}

func (a *Action) newHandlerStruct() Handler {
	return reflect.New(a.handlerStructElemType).Interface().(Handler)
}

func (c *Context) Args() []string {
	return c.args
}

// String makes Author comply to the Stringer interface, to allow an easy print in the templating process
func (a Author) String() string {
	e := ""
	if a.Email != "" {
		e = " <" + a.Email + ">"
	}

	return fmt.Sprintf("%v%v", a.Name, e)
}

func pickCommandAndArguments(arguments []string) (r [2][]string, err error) {
	cmd, args := pickCommand(arguments)
	tidiedArgs, args, err := tidyArgs(args, func(string) (want bool, next bool) { return true, true })
	if err != nil {
		return
	}
	r[0] = append([]string{cmd}, tidiedArgs...)
	if len(args) == 0 {
		return
	}
	cmd, args = pickCommand(args)
	if cmd == "" {
		return r, errors.New("subcommand is empty")
	}
	tidiedArgs, args, err = tidyArgs(args, func(string) (want bool, next bool) { return true, true })
	if err != nil {
		return
	}
	r[1] = append([]string{cmd}, tidiedArgs...)
	return
}

func pickCommand(arguments []string) (string, []string) {
	if len(arguments) > 0 {
		if s := arguments[0]; len(s) > 0 && s[0] != '-' {
			return s, arguments[1:]
		}
	}
	return "", arguments
}
