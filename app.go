package flagx

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/henrylee2cn/ameda"
	"github.com/henrylee2cn/goutil/status"
)

type (
	// App is a application structure. It is recommended that
	// an app be created with the flagx.NewApp() function
	App struct {
		*Command
		appName       string
		version       string
		compiled      time.Time
		authors       []Author
		copyright     string
		notFound      ActionFunc
		usageTemplate *template.Template
		validator     ValidateFunc
		usageText     string
		lock          sync.RWMutex
	}
	// Command a command object
	Command struct {
		app                *App
		parent             *Command
		cmdName            string
		description        string
		filters            []*filterObject
		action             *actionObject
		subcommands        map[string]*Command
		usageBody          string
		usageText          string
		parentUsageVisible bool
		lock               sync.RWMutex
	}
	// ValidateFunc validator for struct flag
	ValidateFunc func(interface{}) error
	// Action action of action
	Action interface {
		// Handle handles arguments.
		// NOTE:
		//  If need to return an error, use *Context.ThrowStatus or *Context.CheckStatus
		Handle(*Context)
	}
	// ActionCopier an interface that can create its own copy
	ActionCopier interface {
		DeepCopy() Action
	}
	// FilterCopier an interface that can create its own copy
	FilterCopier interface {
		DeepCopy() Filter
	}
	// ActionFunc action function
	// NOTE:
	//  If need to return an error, use *Context.ThrowStatus or *Context.CheckStatus
	ActionFunc func(*Context)
	// Filter global options of app
	// NOTE:
	//  If need to return an error, use *Context.ThrowStatus or *Context.CheckStatus
	Filter interface {
		Filter(c *Context, next ActionFunc)
	}
	// FilterFunc filter function
	// NOTE:
	//  If need to return an error, use *Context.ThrowStatus or *Context.CheckStatus
	FilterFunc func(c *Context, next ActionFunc)
	// Context context of an action execution
	Context struct {
		context.Context
		args    []string
		cmdPath []string
		cmd     *Command
	}
	// Author represents someone who has contributed to a cli project.
	Author struct {
		Name  string // The Authors name
		Email string // The Authors email
	}
	// Status a handling status with code, msg, cause and stack.
	Status = status.Status
)

type (
	contextKey    int8
	actionFactory struct {
		elemType reflect.Type
	}
	factory      actionFactory
	actionObject struct {
		cmd           *Command
		flagSet       *FlagSet
		options       map[string]*Flag
		actionFactory ActionCopier
		actionFunc    ActionFunc
	}
	filterObject struct {
		flagSet    *FlagSet
		options    map[string]*Flag
		factory    FilterCopier
		filterFunc FilterFunc
	}
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
	a.SetUsageTemplate(defaultAppUsageTemplate)
	a.Command = newCommand(a, "", "")
	a.SetCmdName("")
	a.SetName("")
	a.SetVersion("")
	a.SetCompiled(time.Time{})
	return a
}

func newCommand(app *App, cmdName, description string) *Command {
	return &Command{
		app:                app,
		cmdName:            cmdName,
		description:        description,
		subcommands:        make(map[string]*Command, 16),
		parentUsageVisible: true, // default
	}
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
//  remove '-' prefix automatically
func (a *App) SetCmdName(cmdName string) {
	a.lock.Lock()
	defer a.lock.Unlock()
	if cmdName == "" {
		cmdName = filepath.Base(os.Args[0])
	}
	a.cmdName = strings.TrimLeft(cmdName, "-")
	a.updateAllUsageLocked()
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
	a.updateAllUsageLocked()
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
	a.updateAllUsageLocked()
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
	a.updateAllUsageLocked()
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
	a.updateAllUsageLocked()
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
	a.updateAllUsageLocked()
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
	a.updateAllUsageLocked()
}

// Handle implements Action interface.
func (fn ActionFunc) Handle(c *Context) {
	fn(c)
}

// Filter implements Filter interface.
func (fn FilterFunc) Filter(c *Context, next ActionFunc) {
	fn(c, next)
}

// SetValidator sets the validation function.
func (a *App) SetValidator(validator ValidateFunc) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.validator = validator
}

// AddSubaction adds a subcommand and its action.
// NOTE:
//  panic when something goes wrong
func (c *Command) AddSubaction(cmdName, description string, action Action, filters ...Filter) {
	c.AddSubcommand(cmdName, description, filters...).SetAction(action)
}

// AddSubcommand adds a subcommand.
// NOTE:
//  panic when something goes wrong
func (c *Command) AddSubcommand(cmdName, description string, filters ...Filter) *Command {
	if c.action != nil {
		panic(fmt.Errorf("action has been set, no subcommand can be set: %q", c.PathString()))
	}
	if cmdName == "" {
		panic("command name is empty")
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.subcommands[cmdName] != nil {
		panic(fmt.Errorf("action named %s already exists", cmdName))
	}
	subCmd := newCommand(c.app, cmdName, description)
	subCmd.parent = c
	for _, filter := range filters {
		subCmd.AddFilter(filter)
	}
	c.subcommands[cmdName] = subCmd
	return subCmd
}

// AddFilter adds the filter action.
// NOTE:
//  if filter is a struct, it can implement the copier interface;
//  panic when something goes wrong
func (c *Command) AddFilter(filter Filter) {
	c.lock.Lock()
	defer c.lock.Unlock()
	var obj filterObject
	obj.flagSet = NewFlagSet(c.cmdName, ContinueOnError|ContinueOnUndefined)

	elemType := ameda.DereferenceType(reflect.TypeOf(filter))
	switch elemType.Kind() {
	case reflect.Struct:
		var ok bool
		obj.factory, ok = filter.(FilterCopier)
		if !ok {
			obj.factory = &factory{elemType: elemType}
		}
		err := obj.flagSet.StructVars(obj.factory.DeepCopy())
		if err != nil {
			panic(err)
		}
		obj.flagSet.VisitAll(func(f *Flag) {
			if obj.options == nil {
				obj.options = make(map[string]*Flag)
			}
			obj.options[f.Name] = f
		})
	case reflect.Func:
		obj.filterFunc = filter.Filter
	}
	c.filters = append(c.filters, &obj)
	c.updateAllUsageLocked()
}

// SetAction sets the action of the command.
// NOTE:
//  if action is a struct, it can implement the copier interface;
//  panic when something goes wrong.
func (c *Command) SetAction(action Action) {
	if len(c.subcommands) > 0 {
		panic(fmt.Errorf("some subcommands have been set, no action can be set: %q", c.PathString()))
	}
	var obj actionObject
	obj.cmd = c
	obj.flagSet = NewFlagSet(c.cmdName, ContinueOnError|ContinueOnUndefined)

	elemType := ameda.DereferenceType(reflect.TypeOf(action))
	switch elemType.Kind() {
	case reflect.Struct:
		var ok bool
		obj.actionFactory, ok = action.(ActionCopier)
		if !ok {
			obj.actionFactory = &actionFactory{elemType: elemType}
		}
		err := obj.flagSet.StructVars(obj.actionFactory.DeepCopy())
		if err != nil {
			panic(err)
		}
		obj.flagSet.VisitAll(func(f *Flag) {
			if obj.options == nil {
				obj.options = make(map[string]*Flag)
			}
			obj.options[f.Name] = f
		})
	case reflect.Func:
		obj.actionFunc = action.Handle
	}
	c.action = &obj
	c.updateAllUsageLocked()
}

// SetNotFound sets the action when the correct command cannot be found.
func (a *App) SetNotFound(fn ActionFunc) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.notFound = fn
}

// SetDefaultValidator sets the default validator of struct flag.
func (a *App) SetDefaultValidator(fn ValidateFunc) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.validator = fn
}

// SetUsageTemplate sets usage template.
func (a *App) SetUsageTemplate(tmpl *template.Template) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.usageTemplate = tmpl
}

// Exec executes the command.
func (c *Command) Exec(ctx context.Context, arguments []string) (stat *Status) {
	defer status.Catch(&stat)
	handle, ctxObj := c.route(ctx, arguments)
	handle(ctxObj)
	return
}

func (c *Command) route(ctx context.Context, arguments []string) (ActionFunc, *Context) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	filters, action, cmdPath, cmd, found := c.findFiltersAndAction([]string{c.cmdName}, arguments)
	actionFunc := action.Handle
	if found {
		for i := len(filters) - 1; i >= 0; i-- {
			filter := filters[i]
			nextHandle := actionFunc
			actionFunc = func(c *Context) {
				filter.Filter(c, nextHandle)
			}
		}
	}
	return actionFunc, &Context{args: arguments, cmdPath: cmdPath, Context: ctx, cmd: cmd}
}

func (c *Command) findFiltersAndAction(cmdPath, arguments []string) ([]Filter, Action, []string, *Command, bool) {
	filters, arguments := c.newFilters(arguments)
	action, arguments, found := c.newAction(arguments)
	if found {
		return filters, action, cmdPath, c, true
	}
	subCmdName, arguments := SplitArgs(arguments)
	subCmd := c.subcommands[subCmdName]
	if subCmdName != "" {
		cmdPath = append(cmdPath, subCmdName)
	}
	if subCmd == nil {
		if c.app.notFound != nil {
			return nil, c.app.notFound, cmdPath, c, false
		}
		ThrowStatus(
			StatusNotFound,
			"",
			fmt.Sprintf("not found command action: %q", strings.Join(cmdPath, " ")),
		)
		return nil, nil, cmdPath, c, false
	}
	subFilters, action, cmdPath, subCmd2, found := subCmd.findFiltersAndAction(cmdPath, arguments)
	if found {
		filters = append(filters, subFilters...)
		return filters, action, cmdPath, subCmd2, true
	}
	return nil, action, cmdPath, subCmd2, false
}

func (c *Command) newFilters(arguments []string) (r []Filter, args []string) {
	r = make([]Filter, len(c.filters))
	args = arguments
	for i, filter := range c.filters {
		if filter.filterFunc != nil {
			r[i] = filter.filterFunc
		} else {
			flagSet := NewFlagSet(c.cmdName, filter.flagSet.ErrorHandling())
			newObj := filter.factory.DeepCopy()
			flagSet.StructVars(newObj)
			err := flagSet.Parse(arguments)
			CheckStatus(err, StatusParseFailed, "")
			if c.app.validator != nil {
				err = c.app.validator(newObj)
			}
			CheckStatus(err, StatusValidateFailed, "")
			r[i] = newObj
			nargs := flagSet.NextArgs()
			if len(args) > len(nargs) {
				args = nargs
			}
		}
	}
	return r, args
}

func (c *Command) newAction(cmdline []string) (Action, []string, bool) {
	a := c.action
	if a == nil {
		return nil, cmdline, false
	}
	cmdName := a.flagSet.Name()
	if a.actionFunc != nil {
		_, cmdline = SplitArgs(cmdline)
		return a.actionFunc, cmdline, true
	}
	flagSet := NewFlagSet(cmdName, a.flagSet.ErrorHandling())
	newObj := a.actionFactory.DeepCopy()
	flagSet.StructVars(newObj)
	err := flagSet.Parse(cmdline)
	CheckStatus(err, StatusParseFailed, "")
	if a.cmd.app.validator != nil {
		err = a.cmd.app.validator(newObj)
	}
	CheckStatus(err, StatusValidateFailed, "")
	return newObj.(Action), flagSet.NextArgs(), true
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

func (h *actionFactory) DeepCopy() Action {
	return reflect.New(h.elemType).Interface().(Action)
}

func (f *factory) DeepCopy() Filter {
	return reflect.New(f.elemType).Interface().(Filter)
}

// CmdName returns the command name of the command.
func (c *Command) CmdName() string {
	return c.cmdName
}

// Path returns the command path slice.
func (c *Command) Path() (p []string) {
	r := c
	for {
		if r.parent == nil {
			p = append(p, r.cmdName)
			ameda.NewStringSlice(p).Reverse()
			return
		}
		p = append(p, r.cmdName)
		r = r.parent
	}
}

// PathString returns the command path string.
func (c *Command) PathString() string {
	return strings.Join(c.Path(), " ")
}

// Root returns the root command.
// NOTE:
//  returns nil if it does not exist.
func (c *Command) Root() *Command {
	r := c
	for {
		if r.parent == nil {
			return r
		}
		r = r.parent
	}
}

// Parent returns the parent command.
// NOTE:
//  returns nil if it does not exist.
func (c *Command) Parent() *Command {
	return c.parent
}

// LookupSubcommand lookups subcommand by path names.
// NOTE:
//  returns nil if it does not exist.
func (c *Command) LookupSubcommand(pathCmdNames ...string) *Command {
	r := c
	for _, name := range pathCmdNames {
		if name == "" {
			continue
		}
		r = r.subcommands[name]
		if r == nil {
			return nil
		}
	}
	return r
}

// Subcommands returns the subcommands.
func (c *Command) Subcommands() []*Command {
	names := make([]string, 0, len(c.subcommands))
	for name := range c.subcommands {
		names = append(names, name)
	}
	sort.Strings(names)
	cmds := make([]*Command, len(names))
	for i, name := range names {
		cmds[i] = c.subcommands[name]
	}
	return cmds
}

// Filters returns the formal flags.
func (c *Command) Filters() map[string]*Flag {
	if c.action == nil {
		return nil
	}
	return c.action.options
}

// Args returns the command arguments.
func (c *Context) Args() []string {
	return c.args
}

// CmdPath returns the command path slice.
func (c *Context) CmdPath() []string {
	return c.cmdPath
}

// CmdPathString returns the command path string.
func (c *Context) CmdPathString() string {
	return strings.Join(c.CmdPath(), " ")
}

// UsageText returns the command usage.
func (c *Context) UsageText(prefix ...string) string {
	return c.cmd.UsageText(prefix...)
}

// ThrowStatus creates a status with stack, and panic.
func (c *Context) ThrowStatus(code int32, msg string, cause interface{}) {
	panic(status.New(code, msg, cause).TagStack(1))
}

// CheckStatus if err!=nil, create a status with stack, and panic.
// NOTE:
//  If err!=nil and msg=="", error text is set to msg
func (c *Context) CheckStatus(err error, code int32, msg string, whenError ...func()) {
	if err == nil {
		return
	}
	if len(whenError) > 0 && whenError[0] != nil {
		whenError[0]()
	}
	panic(status.New(code, msg, err).TagStack(1))
}

// ParentVisible returns the visibility in parent command usage.
func (c *Command) ParentVisible() bool {
	return c.parentUsageVisible
}

// SetParentVisible sets the visibility in parent command usage.
func (c *Command) SetParentVisible(visible bool) {
	c.parentUsageVisible = visible
}

// UsageText returns the usage text.
func (c *Command) UsageText(prefix ...string) string {
	if len(prefix) > 0 {
		return strings.Replace(c.usageText, "\n", "\n"+prefix[0], -1)
	}
	return c.usageText
}

// defaultAppUsageTemplate is the text template for the Default help topic.
var defaultAppUsageTemplate = template.Must(template.New("appUsage").
	Funcs(template.FuncMap{"join": strings.Join}).
	Parse(`{{if .AppName}}{{.AppName}}{{else}}{{.CmdName}}{{end}}{{if .Version}} - v{{.Version}}{{end}}{{if .Description}}

{{.Description}}{{end}}

USAGE:
  {{.Usage}}{{if len .Authors}}

AUTHOR{{with $length := len .Authors}}{{if ne 1 $length}}S{{end}}{{end}}:
{{range $index, $author := .Authors}}{{if $index}}
{{end}}  {{$author}}{{end}}{{end}}{{if .Copyright}}

COPYRIGHT:
  {{.Copyright}}{{end}}
`))

func (c *Command) updateAllUsageLocked() {
	a := c.app
	a.Command.updateUsageLocked()
	text := a.Command.usageText
	data := map[string]interface{}{
		"AppName":     a.appName,
		"CmdName":     a.cmdName,
		"Version":     a.version,
		"Description": a.description,
		"Authors":     a.authors,
		"Usage":       text,
		"Copyright":   a.copyright,
	}
	var buf bytes.Buffer
	err := a.usageTemplate.Execute(&buf, data)
	if err != nil {
		panic(err)
	}
	a.usageText = strings.Replace(buf.String(), "\n\n\n", "\n\n", -1)
}

func (c *Command) updateUsageLocked() {
	c.usageText, c.usageBody = c.newUsageLocked()
	subcommands := c.Subcommands()
	for _, subCmd := range subcommands {
		subCmd.updateUsageLocked()
		if subCmd.parentUsageVisible {
			c.usageText += subCmd.usageText
		}
	}
}

func (c *Command) newUsageLocked() (text string, body string) {
	var buf bytes.Buffer
	flags := make([]*Flag, 0, len(c.filters)+1)
	for _, filter := range c.filters {
		filter.flagSet.RangeAll(func(f *Flag) {
			flags = append(flags, f)
		})
	}
	if c.action != nil {
		c.action.flagSet.RangeAll(func(f *Flag) {
			flags = append(flags, f)
		})
	}
	fn := newPrintOneDefault(&buf, true)
	for _, f := range flags {
		fn(f)
	}
	body = buf.String()
	if c.parent != nil { // non-global command
		var ellipsis string
		if c.action == nil {
			ellipsis = " ..."
		}
		text = fmt.Sprintf("$%s%s\n  %s\n", c.PathString(), ellipsis, c.description)
		body = strings.Replace(body, "-?", "?", -1)
	} else {
		body = strings.Replace(body, "  -?", "?", -1)
		body = strings.Replace(body, "  -", "-", -1)
		body = strings.Replace(body, "\n    \t", "\n  \t", -1)
	}
	text += body
	return text, body
}

// String makes Author comply to the Stringer interface, to allow an easy print in the templating process
func (a Author) String() string {
	e := ""
	if a.Email != "" {
		e = " <" + a.Email + ">"
	}
	return fmt.Sprintf("%v%v", a.Name, e)
}
