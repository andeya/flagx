package flagx

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/andeya/goutil"
	"github.com/andeya/goutil/status"
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
	// Scope command scope
	Scope int32
	// ValidateFunc validator for struct flag
	ValidateFunc func(interface{}) error
	// Author represents someone who has contributed to a cli project.
	Author struct {
		Name  string // The Authors name
		Email string // The Authors email
	}
	// Status a handling status with code, msg, cause and stack.
	Status = status.Status
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
	// CatchStatus recovers the panic and returns status.
	// NOTE:
	//  Set `realStat` to true if a `Status` type is recovered
	// Example:
	//  var stat *Status
	//  defer CatchStatus(&stat)
	// TYPE:
	//  func CatchStatus(statPtr **Status, realStat ...*bool)
	CatchStatus = status.CatchWithStack
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

// CmdName returns the command name of the application.
// Defaults to filepath.Base(os.Args[0])
func (a *App) CmdName() string {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return a.cmdName
}

// SetCmdName sets the command name of the application.
// NOTE:
//
//	remove '-' prefix automatically
func (a *App) SetCmdName(cmdName string) *App {
	a.lock.Lock()
	defer a.lock.Unlock()
	if cmdName == "" {
		cmdName = filepath.Base(os.Args[0])
	}
	a.cmdName = strings.TrimLeft(cmdName, "-")
	a.updateUsageLocked()
	return a
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
func (a *App) SetName(appName string) *App {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.appName = appName
	a.updateUsageLocked()
	return a
}

// Description returns description the of the application.
func (a *App) Description() string {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return a.description
}

// SetDescription sets description the of the application.
func (a *App) SetDescription(description string) *App {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.description = description
	a.updateUsageLocked()
	return a
}

// Version returns the version of the application.
func (a *App) Version() string {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return a.version
}

// SetVersion sets the version of the application.
func (a *App) SetVersion(version string) *App {
	a.lock.Lock()
	defer a.lock.Unlock()
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "V")
	if version == "" {
		version = "0.0.1"
	}
	a.version = version
	a.updateUsageLocked()
	return a
}

// Compiled returns the compilation date.
func (a *App) Compiled() time.Time {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return a.compiled
}

// SetCompiled sets the compilation date.
func (a *App) SetCompiled(date time.Time) *App {
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
	return a
}

// Authors returns the list of all authors who contributed.
func (a *App) Authors() []Author {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return a.authors
}

// SetAuthors sets the list of all authors who contributed.
func (a *App) SetAuthors(authors []Author) *App {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.authors = authors
	a.updateUsageLocked()
	return a
}

// Copyright returns the copyright of the binary if any.
func (a *App) Copyright() string {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return a.copyright
}

// SetCopyright sets copyright of the binary if any.
func (a *App) SetCopyright(copyright string) *App {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.copyright = copyright
	a.updateUsageLocked()
	return a
}

// SetNotFound sets the action when the correct command cannot be found.
func (a *App) SetNotFound(fn ActionFunc) *App {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.notFound = fn
	return a
}

// SetValidator sets parameter validator for struct action and struct filter.
func (a *App) SetValidator(fn ValidateFunc) *App {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.validator = fn
	return a
}

// SetUsageTemplate sets usage template.
func (a *App) SetUsageTemplate(tmpl *template.Template) *App {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.usageTemplate = tmpl
	return a
}

// SetScopeMatcher sets the scope matching function.
func (a *App) SetScopeMatcher(fn func(cmdScope, execScope Scope) error) *App {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.scopeMatcherFunc = fn
	return a
}

// UsageText returns the usage text by by the executor scope.
// NOTE:
//
//	if @scopes is empty, all command usage are returned.
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

// String makes Author comply to the Stringer interface, to allow an easy print in the templating process
func (a Author) String() string {
	e := ""
	if a.Email != "" {
		e = " <" + a.Email + ">"
	}
	return fmt.Sprintf("%v%v", a.Name, e)
}
