package helpers

import "fmt"

// Exit codes form yankrun's stable contract with scripts and agents.
// They are documented in COMMANDS.md and must not be renumbered.
const (
	ExitOK         = 0   // success
	ExitInternal   = 1   // unexpected internal failure
	ExitUsage      = 2   // bad flags / invalid invocation
	ExitValidation = 3   // bad input values, manifest violation, missing required in non-interactive mode
	ExitNotFound   = 4   // directory, template, or branch not found
	ExitGit        = 5   // git clone / network failure
	ExitCancelled  = 130 // user aborted an interactive prompt
)

// CodedError wraps an error with the process exit code it should map to.
// main's ExitErrHandler reads Code to decide os.Exit; everything else can
// treat it as a normal error.
type CodedError struct {
	Code int
	Err  error
}

func (e *CodedError) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *CodedError) Unwrap() error { return e.Err }

// ExitCode extracts the exit code an error should map to. A plain error is an
// internal failure; nil is success.
func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	if ce, ok := err.(*CodedError); ok {
		return ce.Code
	}
	return ExitInternal
}

func coded(code int, format string, args ...any) *CodedError {
	return &CodedError{Code: code, Err: fmt.Errorf(format, args...)}
}

// UsageErr reports an invalid invocation (missing required flag, bad combination).
func UsageErr(format string, args ...any) *CodedError { return coded(ExitUsage, format, args...) }

// ValidationErr reports bad input values or a manifest/required-value violation.
func ValidationErr(format string, args ...any) *CodedError {
	return coded(ExitValidation, format, args...)
}

// NotFoundErr reports a missing directory, template, or branch.
func NotFoundErr(format string, args ...any) *CodedError { return coded(ExitNotFound, format, args...) }

// GitErr reports a clone or network failure. Pass the underlying error so it
// stays unwrappable.
func GitErr(err error) *CodedError { return &CodedError{Code: ExitGit, Err: err} }

// CancelledErr reports that the user aborted an interactive prompt.
func CancelledErr(format string, args ...any) *CodedError {
	return coded(ExitCancelled, format, args...)
}

// WithCode tags an existing error with an exit code, preserving the message.
func WithCode(code int, err error) error {
	if err == nil {
		return nil
	}
	return &CodedError{Code: code, Err: err}
}
