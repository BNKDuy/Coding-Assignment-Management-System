package ExecUtil

import "errors"

var (
	ErrTimedOut     = errors.New("execution timed out")
	ErrRuntimeError = errors.New("runtime error")
	ErrCompileError = errors.New("compile error")
)
