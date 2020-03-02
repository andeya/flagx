package flagx

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/henrylee2cn/ameda"
	"github.com/henrylee2cn/goutil/status"
)

// Command a command object
type Command struct {
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

func newCommand(app *App, cmdName, description string) *Command {
	return &Command{
		app:                app,
		cmdName:            cmdName,
		description:        description,
		subcommands:        make(map[string]*Command, 16),
		parentUsageVisible: true, // default
	}
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
		obj.actionFunc = action.Execute
	}
	c.action = &obj
	if len(scope) > 0 {
		c.scope = scope[0]
	}
	c.app.execScopeUsageTexts = make(map[Scope]string, len(c.app.execScopeUsageTexts))
	c.bubbleSetScopeCmd(c.scope, nil)
	c.app.updateUsageLocked()
}

func (c *Command) bubbleSetScopeCmd(scope Scope, subcmds []*Command) {
	c.execScopeUsageTexts = make(map[Scope]string, len(c.execScopeUsageTexts))
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
	actionFunc := action.Execute
	if found {
		for i := len(filters) - 1; i >= 0; i-- {
			filter := filters[i]
			nextAction := actionFunc
			actionFunc = func(c *Context) {
				filter.Filter(c, nextAction)
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
