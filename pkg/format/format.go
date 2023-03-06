package format

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/cli/safeexec"
)

type Options struct {
	BaseDir    string
	Checks     []string
	Filetypes  []string
	SkipChecks []string

	FormatGenerated       bool
	FormatSubrepositories bool
}

type ResultSet struct {
	Results           []Result `json:"checks"`
	ModifiedFileNames []string `json:"modifiedFileNames"`
	ViolationsCount   int      `json:"violationsCount"`
}

type Result struct {
	Name       string      `json:"name"`
	Violations []Violation `json:"violations"`
}

type Violation struct {
	FileName   string `json:"fileName"`
	Message    string `json:"message"`
	LineNumber int    `json:"lineNumber"`
}

func Format(opts *Options, jarFilePath string) (*ResultSet, error) {
	if err := validateOptions(opts); err != nil {
		return nil, err
	}

	tempOutputFile, err := ioutil.TempFile("", "temp-*.json")
	if err != nil {
		return nil, err
	}
	defer tempOutputFile.Close()
	defer os.Remove(tempOutputFile.Name())

	cmdArgs := []string{"-jar", jarFilePath}
	cmdArgs = append(cmdArgs, fmt.Sprintf("source.base.dir=%s", opts.BaseDir))
	cmdArgs = append(cmdArgs, fmt.Sprintf("output.file.name=%s", tempOutputFile.Name()))

	if opts.Checks != nil {
		cmdArgs = append(cmdArgs, fmt.Sprintf("source.check.names=%s", strings.Join(opts.Checks, ",")))
	}
	if opts.Filetypes != nil {
		cmdArgs = append(cmdArgs, fmt.Sprintf("source.file.extensions=%s", strings.Join(opts.Filetypes, ",")))
	}
	if opts.SkipChecks != nil {
		cmdArgs = append(cmdArgs, fmt.Sprintf("skip.check.names=%s", strings.Join(opts.SkipChecks, ",")))
	}

	exe, err := safeexec.LookPath("java")
	if err != nil {
		return nil, err
	}

	// TODO: Redirect standard output and error into buffers for processing
	execCmd := exec.Command(exe, cmdArgs...)
	execCmd.Run()

	data, err := ioutil.ReadFile(tempOutputFile.Name())
	if err != nil {
		return nil, err
	}

	output := ResultSet{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &output); err != nil {
			return nil, err
		}
	}
	return &output, nil
}

func validateOptions(opts *Options) error {
	if opts == nil {
		return fmt.Errorf("format options cannot be nil")
	}
	if opts.BaseDir == "" {
		return fmt.Errorf("specify base directory")
	}
	return nil
}
