package flagx

import (
	"context"
	"reflect"
	"strings"

	"github.com/henrylee2cn/goutil/status"
)

type (
	// Action action of action
	Action interface {
		// Execute executes action.
		// NOTE:
		//  If need to return an error, use *Context.ThrowStatus or *Context.CheckStatus
		Execute(*Context)
	}
	// ActionFunc action function
	// NOTE:
	//  If need to return an error, use *Context.ThrowStatus or *Context.CheckStatus
	ActionFunc func(*Context)
	// ActionCopier an interface that can create its own copy
	ActionCopier interface {
		DeepCopy() Action
	}
	// Filter global options of app
	// NOTE:
	//  If need to return an error, use *Context.ThrowStatus or *Context.CheckStatus
	Filter interface {
		Filter(c *Context, next ActionFunc)
	}
	// FilterCopier an interface that can create its own copy
	FilterCopier interface {
		DeepCopy() Filter
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

// Execute implements Action interface.
func (fn ActionFunc) Execute(c *Context) {
	fn(c)
}

// Filter implements Filter interface.
func (fn FilterFunc) Filter(c *Context, next ActionFunc) {
	fn(c, next)
}

func (h *actionFactory) DeepCopy() Action {
	return reflect.New(h.elemType).Interface().(Action)
}

func (f *factory) DeepCopy() Filter {
	return reflect.New(f.elemType).Interface().(Filter)
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
