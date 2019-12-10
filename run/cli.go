package run

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/mevansam/goutils/logger"
)

type CLI interface {
	GetPipedOutputBuffer() io.Reader
	GetPipedErrorBuffer() io.Reader
	Run(args []string) error
	RunWithEnv(args []string, extraEnvVars []string) error
}

type cli struct {
	executablePath            string
	workingDirectory          string
	outputBuffer, errorBuffer io.Writer

	// Original buffer if pipe was created
	outBuffer, errBuffer         io.Writer
	outPipeWriter, errPipeWriter *io.PipeWriter
}

func NewCLI(
	executablePath string,
	workingDirectory string,
	outputBuffer, errorBuffer io.Writer,
) (CLI, error) {

	var (
		err  error
		info os.FileInfo
	)

	info, err = os.Stat(executablePath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("binary not found at '%s'", executablePath)
	}
	if err != nil {
		return nil, err
	}
	if (info.Mode() & 0111) == 0 {
		return nil, fmt.Errorf("binary at '%s' is not executable", executablePath)
	}

	info, err = os.Stat(workingDirectory)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("working directory not found at '%s'", workingDirectory)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("working directory '%s' is not a directory", workingDirectory)
	}

	return &cli{
		executablePath:   executablePath,
		workingDirectory: workingDirectory,

		outputBuffer: outputBuffer,
		errorBuffer:  errorBuffer,

		outPipeWriter: nil,
		errPipeWriter: nil,

		outBuffer: nil,
		errBuffer: nil,
	}, nil
}

func (c *cli) GetPipedOutputBuffer() io.Reader {

	// save original buffer
	c.outBuffer = c.outputBuffer

	pr, pw := io.Pipe()
	c.outPipeWriter = pw
	c.outputBuffer = io.MultiWriter(c.outBuffer, c.outPipeWriter)
	return pr
}

func (c *cli) GetPipedErrorBuffer() io.Reader {

	// save original buffer
	c.errBuffer = c.errorBuffer

	pr, pw := io.Pipe()
	c.errPipeWriter = pw
	c.errorBuffer = io.MultiWriter(c.errBuffer, c.errPipeWriter)
	return pr
}

func (c *cli) Run(
	args []string,
) error {
	return c.RunWithEnv(args, []string{})
}

func (c *cli) RunWithEnv(
	args []string,
	extraEnvVars []string,
) error {

	command := exec.Command(c.executablePath, args...)
	command.Dir = c.workingDirectory

	command.Env = os.Environ()
	command.Env = append(command.Env, extraEnvVars...)

	command.Stdout = c.outputBuffer
	command.Stderr = c.errorBuffer

	logger.TraceMessage(
		"Running CLI command:\n  cli path: %s\n  args: %# v\n  env: %# v\n  dir: %s\n",
		c.executablePath,
		args,
		command.Env,
		c.workingDirectory,
	)

	err := command.Run()

	// Restore buffers if piped
	if c.outBuffer != nil {
		c.outPipeWriter.Close()
		c.outputBuffer = c.outBuffer

		c.outPipeWriter = nil
		c.outBuffer = nil
	}
	if c.errBuffer != nil {
		c.errPipeWriter.Close()
		c.errorBuffer = c.errBuffer

		c.errPipeWriter = nil
		c.errBuffer = nil
	}

	return err
}
