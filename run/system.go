package run

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/mitchellh/go-homedir"
)

var (
	outputBuffer bytes.Buffer

	shell CLI
	sherr error
)

func GetSystemCLI(cliName string, outputBuffer *bytes.Buffer, errorBuffer *bytes.Buffer) (CLI, error) {

	var (
		err error

		cliPath string
	)

	if cliPath, err = LookupFilePathInSystem(cliName); err != nil {
		return nil, err
	}

	cwd, _ := os.Getwd()
	return NewCLI(cliPath, cwd, outputBuffer, errorBuffer)
}

func LookupFilePathInSystem(fileName string) (string, error) {

	var (
		err error
	)

	defer outputBuffer.Reset()

	if sherr == nil {
		if runtime.GOOS == "darwin" || runtime.GOOS == "linux" || runtime.GOOS == "openbsd" {
			if err = shell.Run([]string{"-c", fmt.Sprintf("which %s", fileName)}); err != nil {
				return "", fmt.Errorf(
					"Error looking up file '%s' in system path.",
					fileName,
				)
			}
		} else if runtime.GOOS == "windows" {
			// TODO: This needs to be validated
			if err = shell.Run([]string{"/C", fmt.Sprintf("where %s", fileName)}); err != nil {
				return "", fmt.Errorf(
					"Error looking up file '%s' in system path: %s",
					fileName, strings.TrimSuffix(outputBuffer.String(), "\n"),
				)
			}
		}
		return strings.TrimSuffix(outputBuffer.String(), "\n"), nil
	}
	return "", sherr
}

func init() {
	home, _ := homedir.Dir()
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" || runtime.GOOS == "openbsd" {
		shell, sherr = NewCLI("/bin/sh", home, &outputBuffer, &outputBuffer)	
	} else if runtime.GOOS == "windows" {
		shell, sherr = NewCLI("C:\\Windows\\System32\\cmd.exe", home, &outputBuffer, &outputBuffer)
	} else {
		sherr = fmt.Errorf("Unsupported OS.")
	}
}
