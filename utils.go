package deb

import (
	"fmt"
	"runtime"
)

// Returns a nice error that the current method is not yet
// implemented. Is useful in TestDrivenDevelopment
func NotYetImplemented() error {
	//try to get parent
	pc, file, line, ok := runtime.Caller(1)

	if ok == false {
		panic("You should not call utils.NotImplementedError() on static context")
	}

	function := runtime.FuncForPC(pc)

	return fmt.Errorf("%s:%d %s is not yet implemented", file, line, function.Name())
}
