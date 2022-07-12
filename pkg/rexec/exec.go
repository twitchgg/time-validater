package rexec

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Executer executer
type Executer struct {
	Name string
	Path string
	Args []string
	cmd  *exec.Cmd
}

// NewExecuter create executer
func NewExecuter(name, path string, args []string) (*Executer, error) {
	var err error
	if path, err = exec.LookPath(path); err != nil {
		return nil, fmt.Errorf("look path failed: %s", err.Error())
	}
	return &Executer{
		Name: name,
		Path: path,
		Args: args,
	}, nil
}

type ExecResult struct {
	Error  error
	Output string
}

// Start start executer
func (e *Executer) Start() chan ExecResult {
	result := make(chan ExecResult, 1)
	e.cmd = exec.Command(e.Path, e.Args...)
	var stdout, stderr []byte
	var errStdout, errStderr error
	stdoutIn, _ := e.cmd.StdoutPipe()
	stderrIn, _ := e.cmd.StderrPipe()
	go func() {
		stdout, errStdout = e.copyAndCapture(os.Stdout, stdoutIn)
	}()
	go func() {
		stderr, errStderr = e.copyAndCapture(os.Stderr, stderrIn)
	}()

	e.cmd.Start()
	err := e.cmd.Wait()
	if err != nil {
		result <- ExecResult{
			Error: fmt.Errorf("cmd.Run() failed with %s", string(stderr)),
		}
		return result
	}

	if errStdout != nil || errStderr != nil {
		outStr, errStr := string(stdout), string(stderr)
		result <- ExecResult{
			Error: fmt.Errorf("out:\n%s\nerr:\n%s", outStr, errStr),
		}
		return result
	}
	result <- ExecResult{
		Output: string(stdout),
		Error:  nil,
	}
	return result
}

// Stop stop executer
func (e *Executer) Stop() error {
	return e.cmd.Process.Kill()
}

func (e *Executer) copyAndCapture(w io.Writer, r io.Reader) ([]byte, error) {
	scanner := bufio.NewScanner(r)
	var builder strings.Builder
	for scanner.Scan() {
		builder.WriteString(scanner.Text())
		builder.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read output failed: %s", err.Error())
	}
	return []byte(builder.String()), nil
}

// Run run executer
func (e *Executer) Run() (string, error) {
	cmd := exec.Command(e.Path, e.Args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return strings.TrimSpace(out.String()), err
	}
	return strings.TrimSpace(out.String()), nil
}
