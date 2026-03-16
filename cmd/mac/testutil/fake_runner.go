package testutil

import (
	"slices"
	"strings"
)

type response struct {
	output string
	err    error
}

// FakeRunner is a test double for the Runner interface.
// It records all calls and returns pre-configured responses.
type FakeRunner struct {
	responses  map[string]response
	whichMap   map[string]bool
	calls      []string
}

// NewFakeRunner constructs a FakeRunner with empty state.
func NewFakeRunner() *FakeRunner {
	return &FakeRunner{
		responses: make(map[string]response),
		whichMap:  make(map[string]bool),
	}
}

// whenBuilder is a fluent builder for configuring responses.
type whenBuilder struct {
	fr  *FakeRunner
	key string
}

// When starts a fluent builder for a Run/RunPassthrough call.
func (f *FakeRunner) When(name string, args ...string) *whenBuilder {
	key := strings.Join(append([]string{name}, args...), " ")
	return &whenBuilder{fr: f, key: key}
}

// WhenShell starts a fluent builder for a RunShell call.
func (f *FakeRunner) WhenShell(command string) *whenBuilder {
	return &whenBuilder{fr: f, key: "shell:" + command}
}

// WhenSudo starts a fluent builder for a RunSudo call.
func (f *FakeRunner) WhenSudo(name string, args ...string) *whenBuilder {
	key := "sudo " + strings.Join(append([]string{name}, args...), " ")
	return &whenBuilder{fr: f, key: key}
}

// Returns configures the response for the preceding When* call.
func (w *whenBuilder) Returns(output string, err error) *FakeRunner {
	w.fr.responses[w.key] = response{output: output, err: err}
	return w.fr
}

// WhenWhich configures the result of Which(name).
func (f *FakeRunner) WhenWhich(name string, result bool) *FakeRunner {
	f.whichMap[name] = result
	return f
}

// Run records the call and returns any configured response.
func (f *FakeRunner) Run(name string, args ...string) (string, error) {
	key := strings.Join(append([]string{name}, args...), " ")
	f.calls = append(f.calls, key)
	if r, ok := f.responses[key]; ok {
		return r.output, r.err
	}
	return "", nil
}

// RunSudo records a "sudo ..." call and returns any configured response.
func (f *FakeRunner) RunSudo(name string, args ...string) (string, error) {
	key := "sudo " + strings.Join(append([]string{name}, args...), " ")
	f.calls = append(f.calls, key)
	if r, ok := f.responses[key]; ok {
		return r.output, r.err
	}
	return "", nil
}

// RunPassthrough records the call and returns any configured error.
func (f *FakeRunner) RunPassthrough(name string, args ...string) error {
	key := strings.Join(append([]string{name}, args...), " ")
	f.calls = append(f.calls, key)
	if r, ok := f.responses[key]; ok {
		return r.err
	}
	return nil
}

// RunShell records the shell command and returns any configured response.
func (f *FakeRunner) RunShell(command string) (string, error) {
	key := "shell:" + command
	f.calls = append(f.calls, key)
	if r, ok := f.responses[key]; ok {
		return r.output, r.err
	}
	return "", nil
}

// Which returns the pre-configured boolean for the given executable name.
func (f *FakeRunner) Which(name string) bool {
	result, ok := f.whichMap[name]
	if !ok {
		return false
	}
	return result
}

// ExpandHome replaces a leading ~/ with /home/testuser/ for deterministic tests.
func (f *FakeRunner) ExpandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		return "/home/testuser/" + path[2:]
	}
	return path
}

// Calls returns every recorded call key in order.
func (f *FakeRunner) Calls() []string {
	return f.calls
}

// CalledWith reports whether a Run/RunPassthrough call was made with the given args.
func (f *FakeRunner) CalledWith(name string, args ...string) bool {
	key := strings.Join(append([]string{name}, args...), " ")
	return slices.Contains(f.calls, key)
}

// CalledWithSudo reports whether a RunSudo call was made with the given args.
func (f *FakeRunner) CalledWithSudo(name string, args ...string) bool {
	key := "sudo " + strings.Join(append([]string{name}, args...), " ")
	return slices.Contains(f.calls, key)
}

// CalledWithShell reports whether RunShell was called with the given command.
func (f *FakeRunner) CalledWithShell(command string) bool {
	return slices.Contains(f.calls, "shell:"+command)
}
