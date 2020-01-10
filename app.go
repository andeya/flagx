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
	"sync"
	"text/template"
	"time"

	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/status"
)

type (
	// App is a application structure. It is recommended that
	// an app be created with the flagx.NewApp() function
	App struct {
		appName       string
		cmdName       string
		description   string
		version       string
		compiled      time.Time
		authors       []Author
		copyright     string
		middlewares   []Middleware
		notFound      HandlerFunc
		actions       map[string]*Action
		sortedActions []*Action
		usageText     string
		lock          sync.RWMutex
	}
	// Author represents someone who has contributed to a cli project.
	Author struct {
		Name  string // The Authors name
		Email string // The Authors email
	}
	// Action a command action
	Action struct {
		flagSet         *FlagSet
		description     string
		usageBody       string
		usageText       string
		handlerElemType reflect.Type
		handlerFunc     HandlerFunc
	}
	// Handler handler of action
	Handler interface {
		Handle(*Context)
	}
	// HandlerFunc handler function
	HandlerFunc func(*Context)
	// Middleware middleware of an action execution
	Middleware func(c *Context, next func(*Context)) error
	// Context context of an action execution
	Context struct {
		context.Context
		args []string
	}
)

// NewApp creates a new application.
func NewApp() *App {
	a := new(App)
	return a.init()
}

func (a *App) init() *App {
	a.SetCmdName("")
	a.SetName("")
	a.SetVersion("")
	a.SetCompiled(time.Time{})
	return a
}

// CmdName returns the command name of the application.
// Defaults to filepath.Base(os.Args[0])
func (a *App) CmdName() string {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return a.cmdName
}

// SetCmdName sets the command name of the application.
// NOTE:
//  Remove - prefix automatically
func (a *App) SetCmdName(cmdName string) {
	a.lock.Lock()
	defer a.lock.Unlock()
	if cmdName == "" {
		cmdName = filepath.Base(os.Args[0])
	}
	a.cmdName = strings.TrimLeft(cmdName, "-")
	a.updateUsageLocked()
}

// Name returns the name(title) of the application.
// Defaults to *App.CmdName()
func (a *App) Name() string {
	a.lock.RLock()
	defer a.lock.RUnlock()
	if a.appName == "" {
		return a.cmdName
	}
	return a.appName
}

// SetName sets the name(title) of the application.
func (a *App) SetName(appName string) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.appName = appName
	a.updateUsageLocked()
}

// Description returns description the of the application.
func (a *App) Description() string {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return a.description
}

// SetDescription sets description the of the application.
func (a *App) SetDescription(description string) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.description = description
	a.updateUsageLocked()
}

// Version returns the version of the application.
func (a *App) Version() string {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return a.version
}

// SetVersion sets the version of the application.
func (a *App) SetVersion(version string) {
	a.lock.Lock()
	defer a.lock.Unlock()
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "V")
	if version == "" {
		version = "0.0.1"
	}
	a.version = version
	a.updateUsageLocked()
}

// Compiled returns the compilation date.
func (a *App) Compiled() time.Time {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return a.compiled
}

// SetCompiled sets the compilation date.
func (a *App) SetCompiled(date time.Time) {
	a.lock.Lock()
	defer a.lock.Unlock()
	if date.IsZero() {
		info, err := os.Stat(os.Args[0])
		if err != nil {
			date = time.Now()
		} else {
			date = info.ModTime()
		}
	}
	a.compiled = date
	a.updateUsageLocked()
}

// Authors returns the list of all authors who contributed.
func (a *App) Authors() []Author {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return a.authors
}

// SetAuthors sets the list of all authors who contributed.
func (a *App) SetAuthors(authors []Author) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.authors = authors
	a.updateUsageLocked()
}

// Copyright returns the copyright of the binary if any.
func (a *App) Copyright() string {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return a.copyright
}

// SetCopyright sets copyright of the binary if any.
func (a *App) SetCopyright(copyright string) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.copyright = copyright
	a.updateUsageLocked()
}

// Use uses a middleware.
func (a *App) Use(mw Middleware) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.middlewares = append(a.middlewares, mw)
}

// Handle implementes Handler interface.
func (fn HandlerFunc) Handle(c *Context) {
	fn(c)
}

// MustSetOptions sets the global options actions.
func (a *App) MustSetOptions(handler Handler) {
	err := a.SetOptions(handler)
	if err != nil {
		panic(err)
	}
}

// SetOptions sets the global options.
// NOTE:
//  Panic when something goes wrong.
func (a *App) SetOptions(handler Handler) error {
	return a.regAction("", "", handler)
}

// SetNotFound sets the handler when the correct command cannot be found.
func (a *App) SetNotFound(fn HandlerFunc) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.notFound = fn
}

// MustAddAction adds an action.
// NOTE:
//  Panic when something goes wrong.
func (a *App) MustAddAction(cmdName, desc string, handler Handler) {
	err := a.AddAction(cmdName, desc, handler)
	if err != nil {
		panic(err)
	}
}

// AddAction adds an action.
func (a *App) AddAction(cmdName, desc string, handler Handler) error {
	if cmdName == "" {
		return errors.New("action name can not be empty")
	}
	return a.regAction(cmdName, desc, handler)
}

func (a *App) regAction(cmdName, desc string, handler Handler) error {
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.actions[cmdName] != nil {
		return fmt.Errorf("an action named %s already exists", cmdName)
	}
	action, err := newAction(cmdName, desc, handler)
	if err != nil {
		return err
	}
	if a.actions == nil {
		a.actions = make(map[string]*Action)
	}
	a.actions[cmdName] = action
	a.updateUsageLocked()
	return nil
}

// Actions returns the sorted list of all actions.
func (a *App) Actions() []*Action {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return a.sortedActions
}

// Exec executes application based on the arguments.
func (a *App) Exec(ctx context.Context, arguments []string) (stat *status.Status) {
	defer status.Catch(&stat)
	handle := a.route(arguments)
	handle(&Context{args: arguments, Context: ctx})
	return nil
}

func (a *App) route(arguments []string) HandlerFunc {
	a.lock.RLock()
	defer a.lock.RUnlock()
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
			if a.notFound != nil {
				return a.notFound
			}
			if subcommand == "" {
				status.Throw(1, "not support global flags", nil)
			}
			status.Throw(2, fmt.Sprintf("subcommand %q is not defined", subcommand), nil)
		}
		actions = append(actions, action)
	}
	handlerFunc := func(c *Context) {
		for _, action := range actions {
			action.exec(c)
		}
	}
	for i := len(a.middlewares) - 1; i >= 0; i-- {
		middleware := a.middlewares[i]
		nextHandle := handlerFunc
		handlerFunc = func(c *Context) {
			middleware(c, nextHandle)
		}
	}
	return handlerFunc
}

// UsageText returns the usage text.
func (a *App) UsageText() string {
	if a.CmdName() == "" { // not initialized with flagx.NewApp()
		a.init()
	}
	a.lock.RLock()
	defer a.lock.RUnlock()
	return a.usageText
}

func newAction(cmdName, desc string, handler Handler) (*Action, error) {
	var action Action

	action.description = desc
	action.flagSet = NewFlagSet(cmdName, ContinueOnError|ContinueOnUndefined)
	action.handlerElemType = goutil.DereferenceType(reflect.TypeOf(handler))

	switch action.handlerElemType.Kind() {
	case reflect.Struct:
		err := action.flagSet.StructVars(handler)
		if err != nil {
			return nil, err
		}
	case reflect.Func:
		action.handlerFunc = handler.Handle
	}

	// initializate usage
	var buf bytes.Buffer
	action.flagSet.SetOutput(&buf)
	action.flagSet.PrintDefaults()
	action.usageBody = buf.String()
	if cmdName != "" { // non-global command
		action.usageText += fmt.Sprintf("%s # %s\n", cmdName, desc)
	}
	action.usageText += action.usageBody
	action.flagSet.SetOutput(ioutil.Discard)
	return &action, nil
}

// UsageText returns the usage text.
func (a *Action) UsageText() string {
	return a.usageText
}

// CmdName returns the command name of the action.
func (a *Action) CmdName() string {
	return a.flagSet.Name()
}

// Description returns description the of the action.
func (a *Action) Description() string {
	return a.description
}

func (a *Action) exec(c *Context) {
	flagSet := NewFlagSet(a.flagSet.Name(), a.flagSet.ErrorHandling())
	if a.handlerFunc != nil {
		a.handlerFunc(c)
	} else {
		newObj := reflect.New(a.handlerElemType).Interface()
		flagSet.StructVars(newObj)
		newObj.(Handler).Handle(c)
	}
}

// Args returns the arguments.
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

// appUsageTemplate is the text template for the Default help topic.
var appUsageTemplate = template.Must(template.New("appUsage").
	Funcs(template.FuncMap{"join": strings.Join}).
	Parse(`{{if .AppName}}{{.AppName}}{{else}}{{.CmdName}}{{end}}{{if .Version}} - v{{.Version}}{{end}}{{if .Description}}

{{.Description}}{{end}}

USAGE:
   {{.CmdName}}{{if .Options}} [-globaloptions --]{{end}}{{if len .Commands}} [command] [-commandoptions]

COMMANDS:{{range .Commands}}
$ {{$.CmdName}} {{.UsageText}}{{end}}{{end}}{{if .Options}}

GLOBAL OPTIONS:
{{.Options.UsageText}}{{end}}{{if len .Authors}}

AUTHOR{{with $length := len .Authors}}{{if ne 1 $length}}S{{end}}{{end}}:
{{range $index, $author := .Authors}}{{if $index}}
{{end}}{{$author}}{{end}}{{end}}{{if .Copyright}}

COPYRIGHT:
   {{.Copyright}}{{end}}
`))

func (a *App) updateUsageLocked() {
	var data = map[string]interface{}{
		"AppName":     a.appName,
		"CmdName":     a.cmdName,
		"Version":     a.version,
		"Description": a.description,
		"Authors":     a.authors,
		"Commands":    []*Action{},
		"Copyright":   a.copyright,
	}
	if len(a.actions) > 0 {
		nameList := make([]string, 0, len(a.actions))
		a.sortedActions = make([]*Action, 0, len(a.actions))
		for name := range a.actions {
			nameList = append(nameList, name)
		}
		sort.Strings(nameList)
		if nameList[0] == "" {
			g := a.actions[nameList[0]]
			data["Options"] = g
			nameList = nameList[1:]
			a.sortedActions = append(a.sortedActions, g)
		}
		if len(nameList) > 0 {
			actions := make([]*Action, 0, len(nameList))
			for _, name := range nameList {
				g := a.actions[name]
				actions = append(actions, g)
				a.sortedActions = append(a.sortedActions, g)
			}
			data["Commands"] = actions
		}
	}

	var buf bytes.Buffer
	err := appUsageTemplate.Execute(&buf, data)
	if err != nil {
		panic(err)
	}
	a.usageText = strings.Replace(buf.String(), "\n\n\n", "\n\n", -1)
}
