package errors

import (
	"fmt"
	"runtime"
	"strings"
)

func frameFromRuntime(f runtime.Frame) StackFrame {
	fn := f.Function // e.g. "github.com/myapp/users.(*Service).Create"

	var packageName, structName, funcName string

	// Extract short package name (last path segment before first dot)
	shortFn := fn
	if i := strings.LastIndex(fn, "/"); i >= 0 {
		shortFn = fn[i+1:]
	}
	if i := strings.Index(shortFn, "."); i >= 0 {
		packageName = shortFn[:i]
		rest := shortFn[i+1:] // "(*Service).Create" or "Create"

		if strings.HasPrefix(rest, "(") {
			// Has a receiver: "(*Service).Create" or "(Service).Create"
			end := strings.Index(rest, ")")
			if end >= 0 {
				structName = strings.TrimLeft(rest[1:end], "*") // strip * from pointer receivers
				if dot := strings.Index(rest[end:], "."); dot >= 0 {
					funcName = rest[end+dot+1:] // "Create"
				}
			}
		} else {
			// Plain function: "Create"
			funcName = rest
		}
	}

	return StackFrame{
		PackageName: packageName,
		StructName:  structName,
		FuncName:    funcName,
	}
}

func getCallerFrame() StackFrame {
	pcs := make([]uintptr, 1)
	// skip: 0=Callers, 1=getCallerFrame, 2=caller of getCallerFrame
	runtime.Callers(3, pcs)

	frames := runtime.CallersFrames(pcs)
	f, _ := frames.Next()

	return frameFromRuntime(f)
}

func argsToMap(args ...any) map[string]any {
	result := make(map[string]any)

	for i := 0; i < len(args); i++ {
		switch v := args[i].(type) {
		case string:
			// key-value pair: string key followed by a value
			if i+1 < len(args) {
				result[v] = args[i+1]
				i++ // skip the value
			} else {
				// dangling key with no value
				result[v] = nil
			}
		default:
			// unknown type, store with index as key
			result[fmt.Sprintf("arg%d", i)] = v
		}
	}

	return result
}
