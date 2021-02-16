package executor

import "github.com/Waziup/wazigate-edge/edge"

func init() {
	edge.ScriptExecutors["application/javascript"] = JavaScriptExecutor{}
}

func (JavaScriptExecutor) ExecutorName() string {
	return "JavaScript"
}

type JavaScriptExecutor struct{}
