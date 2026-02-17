package Types

import (
	"server/ExecUtil"
)

var SupportedLanguage = map[string]Runner{
	"c":    ExecUtil.RunC,
	"cpp":  ExecUtil.RunCPP,
	"py":   ExecUtil.RunPython,
	"java": ExecUtil.RunJava,
	"go":   ExecUtil.RunGo,
}
