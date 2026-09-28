package exitcode

import (
	"fmt"
)

// ExitStatus is a non-zero status code resulting from running a shell node.
type ExitStatus uint8

const ExitStatusOK ExitStatus = 0

func (s ExitStatus) Error() string { return fmt.Sprintf("exit status %d", s) }

// NewExitStatus creates an error which contains the specified exit status code.
func NewExitStatus(status uint8) error {
	return ExitStatus(status)
}
