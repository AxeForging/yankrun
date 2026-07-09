// Package schema defines yankrun's machine-readable output contract. Every
// command invoked with --json prints exactly one Envelope to stdout. The shape
// is versioned so agents and scripts can depend on it across releases.
package schema

import (
	"encoding/json"
	"io"

	"github.com/AxeForging/yankrun/helpers"
)

// Version is the envelope schema version. Bump it only on breaking changes to
// the envelope or the data payloads it carries.
const Version = 1

// Envelope wraps every --json response. On success Data holds the command's
// payload (a workflow.Summary, ApplyResult, etc.); on failure Error is set.
type Envelope struct {
	SchemaVersion int         `json:"schemaVersion"`
	Command       string      `json:"command"`
	OK            bool        `json:"ok"`
	Data          interface{} `json:"data,omitempty"`
	Error         *ErrorBody  `json:"error,omitempty"`
}

// ErrorBody carries the exit code (the same taxonomy as the process exit code)
// and a human-readable message.
type ErrorBody struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Emit writes a single JSON envelope for command to w. When err is non-nil it
// emits a failure envelope and returns a CodedError so the caller's return
// still drives the process exit code; the data is ignored in that case.
func Emit(w io.Writer, command string, data interface{}, err error) error {
	env := Envelope{SchemaVersion: Version, Command: command, OK: err == nil}
	if err != nil {
		code := helpers.ExitCode(err)
		env.Error = &ErrorBody{Code: code, Message: err.Error()}
	} else {
		env.Data = data
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if encErr := enc.Encode(env); encErr != nil {
		return encErr
	}
	if err != nil {
		// Preserve the exit code, but stay silent on stderr — the envelope on
		// stdout already carries the error for machine consumers.
		return &helpers.CodedError{Code: helpers.ExitCode(err), Err: nil}
	}
	return nil
}
