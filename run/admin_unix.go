// +build linux darwin

package run

import (
	"io"
	"os"
)

func IsAdmin() (bool, error) {
	return os.Geteuid() == 0, nil
}

func RunAsAdmin(outputBuffer, errorBuffer io.Writer) error {
	return RunAsAdminWithArgs(os.Args, outputBuffer, errorBuffer)
}

func RunAsAdminWithArgs(cmdArgs []string, outputBuffer, errorBuffer io.Writer) error {

	var (
		err error
		cli CLI

		workingDirectory string
	)

	if workingDirectory, err = os.Getwd(); err != nil {
		return nil
	}
	if cli, err = NewCLI(
		"/usr/bin/sudo", 
		workingDirectory,
		outputBuffer,
		errorBuffer,
	); err != nil {
		return err
	}
	args := []string{ "-s", "-E" }
	args = append(args, cmdArgs...)
	return cli.Run(args)
}
