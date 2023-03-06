package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/kevhlee/liferay-source-formatter-cli/internal/nexus"
	"github.com/kevhlee/liferay-source-formatter-cli/pkg/format"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

// TODO: Fetch the latest version.
// TODO: Add CLI flags for SF arguments.
// TODO: Vaildate CLI flags.

const (
	SourceFormatterJarVersion = "1.0.2"
	BuildVersion              = "0.0.1"
)

func main() {
	opts := &format.Options{}

	cmd := &cobra.Command{
		Use:     "source-format [directory]",
		Short:   "Liferay Source Formatter CLI",
		Long:    "Run Liferay Source Formatter as a CLI.",
		Version: BuildVersion,
		Example: heredoc.Doc(`
            # Run SF on directory '~/code'
            $ source-format ~/code

            # Run only 'GradleDependenciesCheck' and 'GradleImportsCheck'
            $ source-format --only="GradleDependenciesCheck,GradleImportsCheck"

            # Run only on Java and XML files
            $ source-format --filetypes="java,xml"

            # Skip running 'JavaStylingCheck'
            $ source-format --skip="JavaStylingCheck"
        `),
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return fmt.Errorf("too many arguments")
			}
			if len(args) == 1 {
				baseDir := args[0]
				if _, err := os.Stat(baseDir); err != nil {
					if os.IsNotExist(err) {
						return fmt.Errorf("%s does not exist", baseDir)
					}
					return err
				}
			}
			return nil
		},
		SilenceErrors: true,
		SilenceUsage:  true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: false,
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.BaseDir = "./"
			if len(args) > 0 {
				opts.BaseDir = args[0]
			}
			return run(opts)
		},
	}

	flags := cmd.Flags()
	flags.StringSliceVar(&opts.Checks, "only", nil, "Only run specified checks")
	flags.StringSliceVar(&opts.Filetypes, "filetypes", nil, "Run checks only specified filetypes")
	flags.BoolVar(&opts.FormatGenerated, "generated", false, "Format generated files")
	flags.BoolVar(&opts.FormatSubrepositories, "subrepositories", false, "Format subrepositories")
	flags.StringSliceVar(&opts.SkipChecks, "skip", nil, "Skip specified checks")

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func run(opts *format.Options) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	localShareDir := filepath.Join(home, ".local/share/liferay")
	if err := os.MkdirAll(localShareDir, 0700); err != nil {
		return err
	}

	jarFileName := fmt.Sprintf("com.liferay.source.formatter-%s.jar", SourceFormatterJarVersion)
	jarFilePath := filepath.Join(localShareDir, jarFileName)

	if _, err := os.Stat(jarFilePath); err != nil {
		req, err := http.NewRequest(
			http.MethodGet,
			nexus.GetLiferayJarFileUrl("com.liferay.source.formatter.standalone", SourceFormatterJarVersion),
			nil,
		)
		if err != nil {
			return err
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("received HTTP status code: %s", res.Status)
		}

		file, err := os.OpenFile(jarFilePath, os.O_CREATE|os.O_WRONLY, 0700)
		if err != nil {
			return err
		}

		progress := progressbar.NewOptions64(
			res.ContentLength,
			progressbar.OptionSetDescription(fmt.Sprintf("Installing %s", jarFileName)),
			progressbar.OptionSetWriter(os.Stderr),
			progressbar.OptionShowBytes(true),
			progressbar.OptionSetWidth(16),
			progressbar.OptionShowCount(),
			progressbar.OptionSetPredictTime(false),
			progressbar.OptionClearOnFinish(),
			progressbar.OptionSpinnerType(14),
			progressbar.OptionSetRenderBlankState(true),
		)

		_, err = io.Copy(io.MultiWriter(file, progress), res.Body)
		if err != nil {
			return err
		}
	}

	resultSet, err := format.Format(opts, jarFilePath)
	if err != nil {
		return err
	}
	return displayResults(resultSet)
}

func displayResults(resultSet *format.ResultSet) error {
	if resultSet.ViolationsCount == 0 {
		fmt.Println("No violations found 🌱")
		return nil
	}

	fmt.Printf("Number of violations: %d\n\n", resultSet.ViolationsCount)

	// TODO: Add colors and icons
	for _, check := range resultSet.Results {
		fmt.Printf("Check: %s\n", check.Name)
		for _, violation := range check.Violations {
			if violation.LineNumber == -1 {
				fmt.Printf("\t%s: %s\n", violation.FileName, violation.Message)
			} else {
				fmt.Printf("\t%s: %s (line: %d)\n", violation.FileName, violation.Message, violation.LineNumber)
			}
		}
		return fmt.Errorf("SF issues found.")
	}

	return nil
}
