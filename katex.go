package katex

import (
	_ "embed"
	"fmt"
	"io"
	"runtime"
	"sync"

	"github.com/lithdew/quickjs"
)

//go:embed katex.min.js
var code string

var jsMutex sync.Mutex
var jsRuntime quickjs.Runtime
var jsContext *quickjs.Context
var jsInitialized bool

// Initialize shared JS runtime and context
func initJSRuntime() error {

	if jsInitialized { return nil }

	jsRuntime = quickjs.NewRuntime()
	jsContext = jsRuntime.NewContext()

	// Initialize KaTeX
	result, err := jsContext.Eval(code)
	if err != nil {
		jsContext.Free()
		jsRuntime.Free()
		return err
	}
	defer result.Free()

	// Initialize KaTeX's persistent state
	result, err = jsContext.Eval(`
		katexState = { macros: {} };
	`)
	if err != nil {
		jsContext.Free()
		jsRuntime.Free()
		return err
	}
	defer result.Free()

	jsInitialized = true

	return nil
}

// Close shared JS runtime and context
func CloseJSRuntime() {
	if jsInitialized {
		jsContext.Free()
		jsRuntime.Free()
		jsInitialized = false
	}
}

func Render(w io.Writer, src []byte, display bool, throwOnError bool) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	jsMutex.Lock()
	defer jsMutex.Unlock()

	// Initialize JS runtime if needed
	if err := initJSRuntime(); err != nil { return err }

	globals := jsContext.Globals()

	globals.Set("_EqSrc3120", jsContext.String(string(src)))
	result, err := jsContext.Eval(fmt.Sprintf(`katex.renderToString(_EqSrc3120, {
		displayMode: %t,
		throwOnError: %t,
		macros: katexState.macros
	})`, display, throwOnError))

	if err != nil { return err }

	defer result.Free()

	_, err = io.WriteString(w, result.String())

	return err
}
