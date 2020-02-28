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
	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/status"
)

type (
	// App is a application structure. It is recommended that
	// an app be created with the flagx.NewApp() function
	App struct {
		*Command
		appName                 string
		version                 string
		compiled                time.Time
		authors                 []Author
		copyright               string
		notFound                ActionFunc
		usageTemplate           *template.Template
		validator               ValidateFunc
		usageText               string
		execScopeUsageTexts     map[Scope]string
		execScopeUsageTextsLock sync.RWMutex
		scopeMatcherFunc        func(cmdScope, execScope Scope) error
		lock                    sync.RWMutex
	}
	// Command a command object
	Command struct {
		app                     *App
		parent                  *Command
		cmdName                 string
		description             string
		scope                   Scope
		filters                 []*filterObject
		action                  *actionObject
		subcommands             map[string]*Command
		scopeCommandMap         map[Scope][]*Command // commands with actions by scope
		scopeCommands           []*Command           // commands with actions by scope
		usageText               string
		execScopeUsageTexts     map[Scope]string
		execScopeUsageTextsLock sync.RWMutex
		parentUsageVisible      bool
		meta                    map[interface{}]interface{}
		lock                    sync.RWMutex
	}
	// Scope command scope
	Scope int32
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
		args      []string
		cmdPath   []string
		cmd       *Command
		execScope Scope
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

const (
	// InitialScope the default scope
	InitialScope Scope = 0
)

// Status code
const (
	StatusBadArgs        int32 = 1
	StatusNotFound       int32 = 2
	StatusParseFailed    int32 = 3
	StatusValidateFailed int32 = 4
	StatusMismatchScope  int32 = 5
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
	a.Command = newCommand(a, "", "")
	a.SetUsageTemplate(defaultAppUsageTemplate)
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

// Handle implements Action interface.
func (fn ActionFunc) Handle(c *Context) {
	fn(c)
}

// Filter implements Filter interface.
func (fn FilterFunc) Filter(c *Context, next ActionFunc) {
	fn(c, next)
}

// SetMeta sets the command meta.
func (c *Command) SetMeta(key interface{}, val interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.meta == nil {
		c.meta = make(map[interface{}]interface{}, 16)
	}
	c.meta[key] = val
}

// GetMeta gets the command meta.
func (c *Command) GetMeta(key interface{}) interface{} {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.meta[key]
}

// AddSubaction adds a subcommand and its action.
// NOTE:
//  panic when something goes wrong
func (c *Command) AddSubaction(cmdName, description string, action Action, scope ...Scope) {
	c.AddSubcommand(cmdName, description).SetAction(action, scope...)
}

// AddSubcommand adds a subcommand.
// NOTE:
//  panic when something goes wrong
func (c *Command) AddSubcommand(cmdName, description string, filters ...Filter) *Command {
	if cmdName == "" {
		panic("command name is empty")
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.action != nil {
		panic(fmt.Errorf("action has been set, no subcommand can be set: %q", c.PathString()))
	}
	if c.subcommands[cmdName] != nil {
		panic(fmt.Errorf("action named %s already exists", cmdName))
	}
	subCmd := newCommand(c.app, cmdName, description)
	subCmd.parent = c
	subCmd.AddFilter(filters...)
	c.subcommands[cmdName] = subCmd
	return subCmd
}

// AddFilter adds the filter action.
// NOTE:
//  if filter is a struct, it can implement the copier interface;
//  panic when something goes wrong
func (c *Command) AddFilter(filters ...Filter) {
	c.lock.Lock()
	defer c.lock.Unlock()
	for _, filter := range filters {
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
	}
	c.app.updateUsageLocked()
}

// SetAction sets the action of the command.
// NOTE:
//  if action is a struct, it can implement the copier interface;
//  panic when something goes wrong.
func (c *Command) SetAction(action Action, scope ...Scope) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if len(c.subcommands) > 0 {
		panic(fmt.Errorf("some subcommands have been set, no action can be set: %q", c.PathString()))
	}
	if c.action != nil {
		panic(fmt.Errorf("an action have been set: %q", c.PathString()))
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
	if len(scope) > 0 {
		c.scope = scope[0]
	}
	c.bubbleSetScopeCmd(c.scope, nil)
	c.app.updateUsageLocked()
}

func (c *Command) bubbleSetScopeCmd(scope Scope, subcmds []*Command) {
	if c.scopeCommandMap == nil {
		c.scopeCommandMap = make(map[Scope][]*Command, 16)
	}
	cmds := append(subcmds, c)
	c.scopeCommandMap[scope] = cmdsDistinctAndSort(append(c.scopeCommandMap[scope], cmds...))
	c.scopeCommands = cmdsDistinctAndSort(append(c.scopeCommands, cmds...))
	if c.parent != nil {
		c.parent.bubbleSetScopeCmd(scope, cmds)
	}
}

func cmdsDistinctAndSort(cmds []*Command) []*Command {
	m := make(map[*Command]bool, len(cmds))
	for _, cmd := range cmds {
		m[cmd] = true
	}
	cmds = cmds[:0]
	for cmd := range m {
		cmds = append(cmds, cmd)
	}
	sort.Sort(commandList(cmds))
	return cmds
}

type commandList []*Command

// Len is the number of elements in the collection.
func (c commandList) Len() int {
	return len(c)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (c commandList) Less(i, j int) bool {
	return c[i].PathString() < c[j].PathString()
}

// Swap swaps the elements with indexes i and j.
func (c commandList) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

// SetNotFound sets the action when the correct command cannot be found.
func (a *App) SetNotFound(fn ActionFunc) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.notFound = fn
}

// SetValidator sets parameter validator for struct action and struct filter.
func (a *App) SetValidator(fn ValidateFunc) {
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

// SetScopeMatcher sets the scope matching function.
func (a *App) SetScopeMatcher(fn func(cmdScope, execScope Scope) error) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.scopeMatcherFunc = fn
}

// Exec executes the command.
// NOTE:
//  @arguments does not contain the command name;
//  the default value of @scope is 0.
func (c *Command) Exec(ctx context.Context, arguments []string, execScope ...Scope) (stat *Status) {
	defer status.Catch(&stat)
	var s Scope
	if len(execScope) > 0 {
		s = execScope[0]
	}
	handle, ctxObj := c.route(ctx, arguments, s)
	handle(ctxObj)
	return
}

func (c *Command) route(ctx context.Context, arguments []string, execScope Scope) (ActionFunc, *Context) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	filters, action, cmdPath, cmd, found := c.findFiltersAndAction([]string{c.cmdName}, arguments, execScope)
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
	return actionFunc, &Context{args: arguments, cmdPath: cmdPath, Context: ctx, cmd: cmd, execScope: execScope}
}

func (c *Command) findFiltersAndAction(cmdPath, arguments []string, execScope Scope) ([]Filter, Action, []string, *Command, bool) {
	if c.action != nil && c.app.scopeMatcherFunc != nil {
		CheckStatus(c.app.scopeMatcherFunc(c.scope, execScope), StatusMismatchScope, "")
	}
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
	subFilters, action, cmdPath, subCmd2, found := subCmd.findFiltersAndAction(cmdPath, arguments, execScope)
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
			ameda.StringsReverse(p)
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

// FindActionCommands finds list of action commands by the executor scope.
// NOTE:
//  if @scopes is empty, all action commands are returned.
func (c *Command) FindActionCommands(execScope ...Scope) []*Command {
	c.lock.Lock()
	defer c.lock.Unlock()
	fn := c.app.scopeMatcherFunc
	if fn == nil || len(execScope) == 0 {
		return c.scopeCommands
	}
	scope := execScope[0]
	list := make([]*Command, 0, len(c.scopeCommands))
	for s, sc := range c.scopeCommandMap {
		if fn(s, scope) == nil {
			list = append(list, sc...)
		}
	}
	return list
}

// Flags returns the formal flags.
func (c *Command) Flags() map[string]*Flag {
	if c.action == nil {
		return nil
	}
	return c.action.options
}

// Args returns the command arguments.
func (c *Context) Args() []string {
	return c.args
}

// GetCmdMeta gets the command meta.
func (c *Context) GetCmdMeta(key interface{}) interface{} {
	return c.cmd.GetMeta(key)
}

// CmdPath returns the command path slice.
func (c *Context) CmdPath() []string {
	return c.cmdPath
}

// CmdPathString returns the command path string.
func (c *Context) CmdPathString() string {
	return strings.Join(c.CmdPath(), " ")
}

// CmdScope returns the command scope.
func (c *Context) CmdScope() Scope {
	return c.cmd.scope
}

// ExecScope returns the executor scope.
func (c *Context) ExecScope() Scope {
	return c.cmd.scope
}

// UsageText returns the command usage.
func (c *Context) UsageText() string {
	return c.cmd.UsageText(c.execScope)
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

// UsageText returns the usage text by by the executor scope.
// NOTE:
//  if @scopes is empty, all command usage are returned.
func (c *Command) UsageText(execScope ...Scope) string {
	fn := c.app.scopeMatcherFunc
	if len(execScope) == 0 || fn == nil {
		return c.usageText
	}
	scope := execScope[0]
	c.execScopeUsageTextsLock.RLock()
	txt, ok := c.execScopeUsageTexts[scope]
	c.execScopeUsageTextsLock.RUnlock()
	if ok {
		return txt
	}
	c.execScopeUsageTextsLock.Lock()
	defer c.execScopeUsageTextsLock.Unlock()
	txt, ok = c.execScopeUsageTexts[scope]
	if ok {
		return txt
	}
	m := make(map[*Command]bool, len(c.scopeCommands))
	for s, sc := range c.scopeCommandMap {
		if fn(s, scope) == nil {
			for _, cmd := range sc {
				m[cmd] = true
			}
		}
	}
	txt = c.createUsageLocked(m)
	if c.execScopeUsageTexts == nil {
		c.execScopeUsageTexts = make(map[Scope]string, 16)
	}
	c.execScopeUsageTexts[scope] = txt
	return txt
}

// UsageText returns the usage text by by the executor scope.
// NOTE:
//  if @scopes is empty, all command usage are returned.
func (a *App) UsageText(execScope ...Scope) string {
	a.lock.RLock()
	defer a.lock.RUnlock()
	fn := a.scopeMatcherFunc
	if len(execScope) == 0 || fn == nil {
		return a.usageText
	}
	scope := execScope[0]
	a.execScopeUsageTextsLock.RLock()
	txt, ok := a.execScopeUsageTexts[scope]
	a.execScopeUsageTextsLock.RUnlock()
	if ok {
		return txt
	}
	txt = a.createUsageLocked(execScope...)
	if a.execScopeUsageTexts == nil {
		a.execScopeUsageTexts = make(map[Scope]string, 16)
	}
	a.execScopeUsageTexts[scope] = txt
	return txt
}

// defaultAppUsageTemplate is the text template for the Default help topic.
var defaultAppUsageTemplate = template.Must(template.New("appUsage").
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

func (a *App) updateUsageLocked() {
	a.Command.updateUsageLocked()
	text := goutil.Indent(a.Command.UsageText(), "  ")
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
	s := buf.String()
	for {
		a.usageText = strings.Replace(s, "\n\n\n", "\n\n", -1)
		if a.usageText == s {
			return
		}
		s = a.usageText
	}
}

func (a *App) createUsageLocked(execScope ...Scope) string {
	cmdUsageText := a.Command.UsageText(execScope...)
	text := goutil.Indent(cmdUsageText, "  ")
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
	s := buf.String()
	var usageText string
	for {
		usageText = strings.Replace(s, "\n\n\n", "\n\n", -1)
		if usageText == s {
			return usageText
		}
		s = usageText
	}
	return usageText
}

func (c *Command) updateUsageLocked() {
	c.usageText = c.newUsageLocked()
	subcommands := c.Subcommands()
	for _, subCmd := range subcommands {
		subCmd.updateUsageLocked()
		if subCmd.parentUsageVisible {
			c.usageText += subCmd.usageText
		}
	}
}

func (c *Command) createUsageLocked(m map[*Command]bool) string {
	if !m[c] {
		return ""
	}
	usageText := c.newUsageLocked()
	for _, subCmd := range c.Subcommands() {
		if subCmd.parentUsageVisible {
			usageText += subCmd.createUsageLocked(m)
		}
	}
	return usageText
}

func (c *Command) newUsageLocked() (text string) {
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
	body := buf.String()
	if c.parent != nil { // non-global command
		var ellipsis string
		if c.action == nil {
			ellipsis = " ..."
		}
		text = fmt.Sprintf("$%s%s\n  %s\n", c.PathString(), ellipsis, c.description)
	} else {
		body = strings.Replace(body, "  -", "-", -1)
		body = strings.Replace(body, "\n    \t", "\n  \t", -1)
	}
	body = strings.Replace(body, "-?", "?", -1)
	text += body
	return text
}

// String makes Author comply to the Stringer interface, to allow an easy print in the templating process
func (a Author) String() string {
	e := ""
	if a.Email != "" {
		e = " <" + a.Email + ">"
	}
	return fmt.Sprintf("%v%v", a.Name, e)
}
