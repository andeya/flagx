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
		appName          string
		cmdName          string
		description      string
		version          string
		compiled         time.Time
		authors          []Author
		copyright        string
		middlewares      []Middleware
		notFound         HandlerFunc
		actions          map[string]*Action
		sortedActions    []*Action
		usageText        string
		defaultValidator ValidateFunc
		lock             sync.RWMutex
	}
	// ValidateFunc validator for struct flag
	ValidateFunc func(interface{}) error
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
		validateFunc    func(interface{}) error
	}
	// Handler handler of action
	Handler interface {
		Handle(*Context) *Status
	}
	// HandlerFunc handler function
	HandlerFunc func(*Context) *Status
	// Middleware middleware of an action execution
	Middleware func(c *Context, next HandlerFunc) *Status
	// Context context of an action execution
	Context struct {
		context.Context
		argsGroup map[string][]string
	}
	// Status a handling status with code, msg, cause and stack.
	Status     = status.Status
	contextKey int8
)

// Status code
const (
	StatusBadArgs        int32 = 1
	StatusNotFound       int32 = 2
	StatusParseFailed    int32 = 3
	StatusValidateFailed int32 = 4
)

const (
	currCmdName contextKey = iota
)

var (
	// NewStatus creates a message status with code, msg and cause.
	// NOTE:
	//  code=0 means no error
	// TYPE:
	//  func NewStatus(code int32, msg string, cause interface{}) *Status
	NewStatus = status.New

	// NewStatusWithStack creates a message status with code, msg and cause and stack.
	// NOTE:
	//  code=0 means no error
	// TYPE:
	//  func NewStatusWithStack(code int32, msg string, cause interface{}) *Status
	NewStatusWithStack = status.NewWithStack

	// NewStatusFromQuery parses the query bytes to a status object.
	// TYPE:
	//  func NewStatusFromQuery(b []byte, tagStack bool) *Status
	NewStatusFromQuery = status.FromQuery
	// CheckStatus if err!=nil, create a status with stack, and panic.
	// NOTE:
	//  If err!=nil and msg=="", error text is set to msg
	// TYPE:
	//  func Check(err error, code int32, msg string, whenError ...func())
	CheckStatus = status.Check
	// ThrowStatus creates a status with stack, and panic.
	// TYPE:
	//  func Throw(code int32, msg string, cause interface{})
	ThrowStatus = status.Throw
	// PanicStatus panic with stack trace.
	// TYPE:
	//  func Panic(stat *Status)
	PanicStatus = status.Panic
	// CatchStatus recovers the panic and returns status.
	// NOTE:
	//  Set `realStat` to true if a `Status` type is recovered
	// Example:
	//  var stat *Status
	//  defer Catch(&stat)
	// TYPE:
	//  func Catch(statPtr **Status, realStat ...*bool)
	CatchStatus = status.Catch
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
func (fn HandlerFunc) Handle(c *Context) *Status {
	return fn(c)
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
func (a *App) SetOptions(handler Handler, validator ...ValidateFunc) error {
	return a.regAction("", "", handler, validator)
}

// SetNotFound sets the handler when the correct command cannot be found.
func (a *App) SetNotFound(fn HandlerFunc) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.notFound = fn
}

// SetDefaultValidator sets the default validator of struct flag.
func (a *App) SetDefaultValidator(fn ValidateFunc) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.defaultValidator = fn
}

// MustAddAction adds an action.
// NOTE:
//  Panic when something goes wrong.
func (a *App) MustAddAction(cmdName, desc string, handler Handler, validator ...ValidateFunc) {
	err := a.AddAction(cmdName, desc, handler, validator...)
	if err != nil {
		panic(err)
	}
}

// AddAction adds an action.
func (a *App) AddAction(cmdName, desc string, handler Handler, validator ...ValidateFunc) error {
	if cmdName == "" {
		return errors.New("action name can not be empty")
	}
	return a.regAction(cmdName, desc, handler, validator)
}

func (a *App) regAction(cmdName, desc string, handler Handler, validator []ValidateFunc) error {
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.actions[cmdName] != nil {
		return fmt.Errorf("an action named %s already exists", cmdName)
	}

	action, err := newAction(cmdName, desc, handler, append(validator, a.defaultValidator)[0])
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
func (a *App) Exec(ctx context.Context, arguments []string) (stat *Status) {
	defer status.Catch(&stat)
	handle, ctxObj, stat := a.route(ctx, arguments)
	if stat.OK() {
		stat = handle(ctxObj)
	}
	return stat
}

func (a *App) route(ctx context.Context, arguments []string) (HandlerFunc, *Context, *Status) {
	a.lock.RLock()
	defer a.lock.RUnlock()
	argsGroup, err := pickCommandAndOptions(arguments)
	if err != nil {
		return nil, nil, NewStatus(StatusBadArgs, "bad arguments", err)
	}
	var ctxObj = &Context{argsGroup: argsGroup, Context: ctx}
	var actions = make([]*Action, 0, 2)
	var handlerFunc func(c *Context) *Status

	for cmdName := range argsGroup {
		action := a.actions[cmdName]
		if action == nil {
			if a.notFound != nil {
				// middleware is still executed
				handlerFunc = func(c *Context) *Status {
					return a.notFound(c.new(cmdName))
				}
				break
			}
			if cmdName == "" {
				return nil, nil, NewStatus(StatusNotFound, "not support global options", nil)
			}
			return nil, nil, NewStatus(StatusNotFound, "subcommand %q is not defined", nil)
		}
		actions = append(actions, action)
	}
	if handlerFunc == nil {
		handlerFunc = func(c *Context) (stat *Status) {
			for _, action := range actions {
				stat = action.handle(c)
				if !stat.OK() {
					return stat
				}
			}
			return stat
		}
	}
	for i := len(a.middlewares) - 1; i >= 0; i-- {
		middleware := a.middlewares[i]
		nextHandle := handlerFunc
		handlerFunc = func(c *Context) *Status {
			return middleware(c, nextHandle)
		}
	}
	return handlerFunc, ctxObj, nil
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

func newAction(cmdName, desc string, handler Handler, validateFunc func(interface{}) error) (*Action, error) {
	var action Action
	action.validateFunc = validateFunc
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

// Exec executes the action alone.
func (a *Action) Exec(c context.Context, options []string) *Status {
	cmdName := a.flagSet.Name()
	return a.handle(newContext(c, cmdName, map[string][]string{cmdName: options}))
}

func (a *Action) handle(c *Context) *Status {
	cmdName := a.flagSet.Name()
	c = c.new(cmdName)
	if a.handlerFunc != nil {
		return a.handlerFunc(c)
	}
	flagSet := NewFlagSet(cmdName, a.flagSet.ErrorHandling())
	newObj := reflect.New(a.handlerElemType).Interface()
	flagSet.StructVars(newObj)
	err := flagSet.Parse(c.argsGroup[cmdName])
	if err != nil {
		return NewStatus(StatusParseFailed, err.Error(), err)
	}
	if a.validateFunc != nil {
		err = a.validateFunc(newObj)
	}
	if err != nil {
		return NewStatus(StatusValidateFailed, err.Error(), err)
	}
	return newObj.(Handler).Handle(c)
}

func newContext(ctx context.Context, cmdName string, argsGroup map[string][]string) *Context {
	return &Context{
		Context:   context.WithValue(ctx, currCmdName, cmdName),
		argsGroup: argsGroup,
	}
}

func (c *Context) new(cmdName string) *Context {
	return newContext(c.Context, cmdName, c.argsGroup)
}

// CmdName returns the command name.
// NOTE:
//  global command name is ""
func (c *Context) CmdName() string {
	cmdName, _ := c.Context.Value(currCmdName).(string)
	return cmdName
}

// Args returns the current command and options.
// NOTE:
//  global command name is ""
func (c *Context) Args() (cmdName string, options []string) {
	cmdName = c.CmdName()
	options = c.argsGroup[cmdName]
	return cmdName, options
}

// String makes Author comply to the Stringer interface, to allow an easy print in the templating process
func (a Author) String() string {
	e := ""
	if a.Email != "" {
		e = " <" + a.Email + ">"
	}
	return fmt.Sprintf("%v%v", a.Name, e)
}

func pickCommandAndOptions(arguments []string) (r map[string][]string, err error) {
	cmd, args := pickCommand(arguments)
	tidiedArgs, args, err := tidyArgs(args, func(string) (want bool, next bool) { return true, true })
	if err != nil {
		return
	}
	r = make(map[string][]string, 2)
	r[cmd] = tidiedArgs
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
	r[cmd] = tidiedArgs
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
