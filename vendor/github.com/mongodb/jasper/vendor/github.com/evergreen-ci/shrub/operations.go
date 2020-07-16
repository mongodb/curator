package shrub

import (
	"encoding/json"
	"errors"
	"fmt"
)

////////////////////////////////////////////////////////////////////////
//
// Specific Command Implementations

func exportCmd(cmd Command) map[string]interface{} {
	if err := cmd.Validate(); err != nil {
		panic(err)
	}

	jsonStruct, err := json.Marshal(cmd)
	if err == nil {
		out := map[string]interface{}{}
		if err = json.Unmarshal(jsonStruct, &out); err == nil {
			return out
		}
	}

	panic(err)
}

type CmdExec struct {
	Binary                        string            `json:"binary,omitempty" yaml:"binary,omitempty"`
	Args                          []string          `json:"args,omitempty" yaml:"args,omitempty"`
	KeepEmptyArgs                 bool              `json:"keep_empty_args,omitempty" yaml:"keep_empty_args,omitempty"`
	Command                       string            `json:"command,omitempty" yaml:"command,omitempty"`
	ContinueOnError               bool              `json:"continue_on_err,omitempty" yaml:"continue_on_err,omitempty"`
	Background                    bool              `json:"background,omitempty" yaml:"background,omitempty"`
	Silent                        bool              `json:"silent,omitempty" yaml:"silent,omitempty"`
	RedirectStandardErrorToOutput bool              `json:"redirect_standard_error_to_output,omitempty" yaml:"redirect_standard_error_to_output,omitempty"`
	IgnoreStandardError           bool              `json:"ignore_standard_error,omitempty" yaml:"ignore_standard_error,omitempty"`
	IgnoreStandardOutput          bool              `json:"ignore_standard_out,omitempty" yaml:"ignore_standard_out,omitempty"`
	Path                          []string          `json:"add_to_path,omitempty" yaml:"add_to_path,omitempty"`
	Env                           map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	AddExpansionsToEnv            bool              `json:"add_expansions_to_env,omitempty" yaml:"add_expansions_to_env,omitempty"`
	IncludeExpansionsInEnv        []string          `json:"include_expansions_in_env,omitempty" yaml:"include_expansions_in_env,omitempty"`
	SystemLog                     bool              `json:"system_log,omitempty" yaml:"system_log,omitempty"`
	WorkingDirectory              string            `json:"working_dir,omitempty" yaml:"working_dir,omitempty"`
}

func (c CmdExec) Name() string    { return "subprocess.exec" }
func (c CmdExec) Validate() error { return nil }
func (c CmdExec) Resolve() *CommandDefinition {
	return &CommandDefinition{
		CommandName: c.Name(),
		Params:      exportCmd(c),
	}
}
func subprocessExecFactory() Command { return CmdExec{} }

type CmdExecShell struct {
	Script                        string `json:"script" yaml:"script"`
	Shell                         string `json:"shell,omitempty" yaml:"shell,omitempty"`
	ContinueOnError               bool   `json:"continue_on_err,omitempty" yaml:"continue_on_err,omitempty"`
	Background                    bool   `json:"background,omitempty" yaml:"background,omitempty"`
	Silent                        bool   `json:"silent,omitempty" yaml:"silent,omitempty"`
	RedirectStandardErrorToOutput bool   `json:"redirect_standard_error_to_output,omitempty" yaml:"redirect_standard_error_to_output,omitempty"`
	IgnoreStandardError           bool   `json:"ignore_standard_error" yaml:"ignore_standard_error"`
	IgnoreStandardOutput          bool   `json:"ignore_standard_out" yaml:"ignore_standard_out"`
	SystemLog                     bool   `json:"system_log,omitempty" yaml:"system_log,omitempty"`
	WorkingDirectory              string `json:"working_dir,omitempty" yaml:"working_dir,omitempty"`
}

func (c CmdExecShell) Name() string    { return "shell.exec" }
func (c CmdExecShell) Validate() error { return nil }
func (c CmdExecShell) Resolve() *CommandDefinition {
	return &CommandDefinition{
		CommandName: c.Name(),
		Params:      exportCmd(c),
	}
}
func shellExecFactory() Command { return CmdExecShell{} }

type ScriptingTestOptions struct {
	Name        string   `json:"name,omitempty" yaml:"name,omitempty"`
	Args        []string `json:"args,omitempty" yaml:"args,omitempty"`
	Pattern     string   `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	TimeoutSecs int      `json:"timeout_secs,omitempty" yaml:"timeout_secs,omitempty"`
	Count       int      `json:"count,omitempty" yaml:"count,omitempty"`
}

type CmdSubprocessScripting struct {
	Harness                       string                `json:"harness" yaml:"harness"`
	Command                       string                `json:"command,omitempty" yaml:"command,omitempty"`
	Args                          []string              `json:"args,omitempty" yaml:"args,omitempty"`
	TestDir                       string                `json:"test_dir,omitempty" yaml:"test_dir,omitempty"`
	TestOptions                   *ScriptingTestOptions `json:"test_options,omitempty" yaml:"test_options,omitempty"`
	Report                        bool                  `json:"report,omitempty" yaml:"report,omitempty"`
	Script                        string                `json:"script,omitempty" yaml:"script,omitempty"`
	ContinueOnError               bool                  `json:"continue_on_err,omitempty" yaml:"continue_on_err,omitempty"`
	Silent                        bool                  `json:"silent,omitempty" yaml:"silent,omitempty"`
	Path                          []string              `json:"add_to_path,omitempty" yaml:"path,omitempty"`
	Env                           map[string]string     `json:"env,omitempty" yaml:"env,omitempty"`
	AddExpansionsToEnv            bool                  `json:"add_expansions_to_env,omitempty" yaml:"add_expansions_to_env,omitempty"`
	IncludeExpansionsInEnv        []string              `json:"include_expansions_in_env,omitempty" yaml:"include_expansions_in_env,omitempty"`
	RedirectStandardErrorToOutput bool                  `json:"redirect_standard_error_to_output,omitempty" yaml:"redirect_standard_error_to_output,omitempty"`
	IgnoreStandardOutput          bool                  `json:"ignore_standard_out,omitempty" yaml:"ignore_standard_out,omitempty"`
	IgnoreStandardError           bool                  `json:"ignore_standard_error,omitempty" yaml:"ignore_standard_error,omitempty"`
	SystemLog                     bool                  `json:"system_log,omitempty" yaml:"system_log,omitempty"`
	WorkingDir                    string                `json:"working_dir,omitempty" yaml:"working_dir,omitempty"`
	CacheDurationSeconds          int                   `json:"cache_duration_secs,omitempty" yaml:"cache_duration_secs,omitempty"`
	CleanupHarness                bool                  `json:"cleanup_harness,omitempty" yaml:"cleanup_harness,omitempty"`
	LockFile                      string                `json:"lock_file,omitempty" yaml:"lock_file,omitempty"`
	Packages                      []string              `json:"packages,omitempty" yaml:"packages,omitempty"`
	HarnessPath                   string                `json:"harness_path,omitempty" yaml:"harness_path,omitempty"`
	HostPath                      string                `json:"host_path,omitempty" yaml:"host_path,omitempty"`
}

func (c CmdSubprocessScripting) Name() string { return "subprocess.scripting" }

func (c CmdSubprocessScripting) Validate() error {
	return nil
}

func (c CmdSubprocessScripting) Resolve() *CommandDefinition {
	return &CommandDefinition{
		CommandName: c.Name(),
		Params:      exportCmd(c),
	}
}

func subprocessScriptingFactory() Command { return CmdSubprocessScripting{} }

type CmdS3Put struct {
	AWSKey                        string   `json:"aws_key" yaml:"aws_key"`
	AWSSecret                     string   `json:"aws_secret" yaml:"aws_secret"`
	Bucket                        string   `json:"bucket" yaml:"bucket"`
	Region                        string   `json:"region,omitempty" yaml:"region,omitempty"`
	ContentType                   string   `json:"content_type" yaml:"content_type"`
	Permissions                   string   `json:"permissions,omitempty" yaml:"permissions,omitempty"`
	Visibility                    string   `json:"visibility,omitempty" yaml:"visibility,omitempty"`
	LocalFile                     string   `json:"local_file,omitempty" yaml:"local_file,omitempty"`
	LocalFilesIncludeFilter       []string `json:"local_files_include_filter,omitempty" yaml:"local_files_include_filter,omitempty"`
	LocalFilesIncludeFilterPrefix string   `json:"local_files_include_filter_prefix,omitempty" yaml:"local_files_include_filter_prefix,omitempty"`
	RemoteFile                    string   `json:"remote_file" yaml:"remote_file"`
	ResourceDisplayName           string   `json:"display_name,omitempty" yaml:"display_name,omitempty"`
	BuildVariants                 []string `json:"build_variants,omitempty" yaml:"build_variants,omitempty"`
	Optional                      bool     `json:"optional,omitempty" yaml:"optional,omitempty"`
}

func (c CmdS3Put) Name() string { return "s3.put" }
func (c CmdS3Put) Validate() error {
	switch {
	case c.AWSKey == "", c.AWSSecret == "":
		return errors.New("must specify aws credentials")
	case c.LocalFile == "" && len(c.LocalFilesIncludeFilter) == 0:
		return errors.New("must specify a local file to upload")
	default:
		return nil
	}
}
func (c CmdS3Put) Resolve() *CommandDefinition {
	return &CommandDefinition{
		CommandName: c.Name(),
		Params:      exportCmd(c),
	}
}
func s3PutFactory() Command { return CmdS3Put{} }

type CmdS3Get struct {
	AWSKey        string   `json:"aws_key" yaml:"aws_key"`
	AWSSecret     string   `json:"aws_secret" yaml:"aws_secret"`
	RemoteFile    string   `json:"remote_file" yaml:"remote_file"`
	Bucket        string   `json:"bucket" yaml:"bucket"`
	LocalFile     string   `json:"local_file,omitempty" yaml:"local_file,omitempty"`
	ExtractTo     string   `json:"extract_to,omitempty" yaml:"extract_to,omitempty"`
	BuildVariants []string `json:"build_variants,omitempty" yaml:"build_variants,omitempty"`
}

func (c CmdS3Get) Name() string    { return "s3.get" }
func (c CmdS3Get) Validate() error { return nil }
func (c CmdS3Get) Resolve() *CommandDefinition {
	return &CommandDefinition{
		CommandName: c.Name(),
		Params:      exportCmd(c),
	}
}
func s3GetFactory() Command { return CmdS3Get{} }

type CmdS3Copy struct {
	AWSKey    string `json:"aws_key" yaml:"aws_key"`
	AWSSecret string `json:"aws_secret" yaml:"aws_secret"`
	Files     []struct {
		Source struct {
			Bucket string `json:"bucket" yaml:"bucket"`
			Path   string `json:"path" yaml:"path"`
			Region string `json:"region,omitempty" yaml:"region,omitempty"`
		} `json:"source" yaml:"source"`
		Destination struct {
			Bucket string `json:"bucket" yaml:"bucket"`
			Path   string `json:"path" yaml:"path"`
			Region string `json:"region,omitempty" yaml:"region,omitempty"`
		} `json:"destination" yaml:"destination"`
		DisplayName   string   `json:"display_name,omitempty" yaml:"display_name,omitempty"`
		Permissions   string   `json:"permissions,omitempty" yaml:"permissions,omitempty"`
		BuildVariants []string `json:"build_variants,omitempty" yaml:"build_variants,omitempty"`
		Optional      bool     `json:"optional,omitempty" yaml:"optional,omitempty"`
	} `json:"s3_copy_files" yaml:"s3_copy_files"`
}

func (c CmdS3Copy) Name() string    { return "s3Copy.copy" }
func (c CmdS3Copy) Validate() error { return nil }
func (c CmdS3Copy) Resolve() *CommandDefinition {
	return &CommandDefinition{
		CommandName: c.Name(),
		Params:      exportCmd(c),
	}
}
func s3CopyFactory() Command { return CmdS3Copy{} }

type CmdS3Push struct {
	ExcludeFilter string `json:"exclude,omitempty" yaml:"exclude,omitempty"`
	MaxRetries    int    `json:"max_retries,omitempty" yaml:"max_retries,omitempty"`
}

func (c CmdS3Push) Name() string    { return "s3.push" }
func (c CmdS3Push) Validate() error { return nil }
func (c CmdS3Push) Resolve() *CommandDefinition {
	return &CommandDefinition{
		CommandName: c.Name(),
		Params:      exportCmd(c),
	}
}
func s3PushFactory() Command { return CmdS3Push{} }

type CmdS3Pull struct {
	ExcludeFilter string `json:"exclude,omitempty" yaml:"exclude,omitempty"`
	MaxRetries    int    `json:"max_retries,omitempty" yaml:"max_retries,omitempty"`
}

func (c CmdS3Pull) Name() string    { return "s3.pull" }
func (c CmdS3Pull) Validate() error { return nil }
func (c CmdS3Pull) Resolve() *CommandDefinition {
	return &CommandDefinition{
		CommandName: c.Name(),
		Params:      exportCmd(c),
	}
}
func s3PullFactory() Command { return CmdS3Pull{} }

type CmdGetProject struct {
	Directory string            `json:"directory" yaml:"directory"`
	Token     string            `json:"token,omitempty" yaml:"token,omitempty"`
	Revisions map[string]string `json:"revisions,omitempty" yaml:"revisions,omitempty"`
}

func (c CmdGetProject) Name() string    { return "git.get_project" }
func (c CmdGetProject) Validate() error { return nil }
func (c CmdGetProject) Resolve() *CommandDefinition {
	return &CommandDefinition{
		CommandName: c.Name(),
		Params:      exportCmd(c),
	}
}
func getProjectFactory() Command { return CmdGetProject{} }

type CmdResultsJSON struct {
	File string `json:"file_location" yaml:"file_location"`
}

func (c CmdResultsJSON) Name() string    { return "attach.results" }
func (c CmdResultsJSON) Validate() error { return nil }
func (c CmdResultsJSON) Resolve() *CommandDefinition {
	return &CommandDefinition{
		CommandName: c.Name(),
		Params:      exportCmd(c),
	}
}
func jsonResultsFactory() Command { return CmdResultsJSON{} }

type CmdResultsXunit struct {
	File  string   `json:"file,omitempty" yaml:"file,omitempty"`
	Files []string `json:"files,omitempty" yaml:"files,omitempty"`
}

func (c CmdResultsXunit) Name() string    { return "attach.xunit_results" }
func (c CmdResultsXunit) Validate() error { return nil }
func (c CmdResultsXunit) Resolve() *CommandDefinition {
	return &CommandDefinition{
		CommandName: c.Name(),
		Params:      exportCmd(c),
	}
}
func xunitResultsFactory() Command { return CmdResultsXunit{} }

type CmdResultsGoTest struct {
	JSONFormat   bool     `json:"-" yaml:"-"`
	LegacyFormat bool     `json:"-" yaml:"-"`
	Files        []string `json:"files" yaml:"files"`
}

func (c CmdResultsGoTest) Name() string {
	if c.LegacyFormat {
		return "gotest.parse_files"
	}
	return "gotest.parse_json"
}
func (c CmdResultsGoTest) Validate() error {
	if c.JSONFormat == c.LegacyFormat {
		return errors.New("invalid format for gotest operation")
	}

	return nil
}
func (c CmdResultsGoTest) Resolve() *CommandDefinition {
	if c.JSONFormat {
		return &CommandDefinition{
			CommandName: c.Name(),
			Params:      exportCmd(c),
		}
	}

	return &CommandDefinition{
		CommandName: "gotest.parse_files",
		Params:      exportCmd(c),
	}
}
func goTestResultsFactory() Command { return CmdResultsGoTest{} }

type ArchiveFormat string

const (
	ZIP     ArchiveFormat = "zip"
	TARBALL ArchiveFormat = "tarball"
)

func (f ArchiveFormat) Validate() error {
	switch f {
	case ZIP, TARBALL:
		return nil
	default:
		return fmt.Errorf("'%s' is not a valid archive format", f)
	}
}

func (f ArchiveFormat) createCmdName() string {
	switch f {
	case ZIP:
		return "archive.zip_pack"
	case TARBALL:
		return "archive.targz_pack"
	default:
		panic(f.Validate())
	}
}

func (f ArchiveFormat) extractCmdName() string {
	switch f {
	case ZIP:
		return "archive.zip_extract"
	case TARBALL:
		return "archive.targz_extract"
	case "auto":
		return "archive.auto_extract"
	default:
		panic(f.Validate())
	}

}

type CmdArchiveCreate struct {
	Format       ArchiveFormat `json:"-" yaml:"-"`
	Target       string        `json:"target" yaml:"target"`
	SourceDir    string        `json:"source_dir" yaml:"source_dir"`
	Include      []string      `json:"include" yaml:"include"`
	ExcludeFiles []string      `json:"exclude_files" yaml:"exclude_files"`
}

func (c CmdArchiveCreate) Name() string    { return c.Format.createCmdName() }
func (c CmdArchiveCreate) Validate() error { return c.Format.Validate() }
func (c CmdArchiveCreate) Resolve() *CommandDefinition {
	return &CommandDefinition{
		CommandName: c.Name(),
		Params:      exportCmd(c),
	}
}

type CmdArchiveExtract struct {
	Format          ArchiveFormat `json:"-" yaml:"-"`
	ArchivePath     string        `json:"path" yaml:"path"`
	TargetDirectory string        `json:"destination,omitempty" yaml:"destination,omitempty"`
	Exclude         []string      `json:"exclude_files,omitempty" yaml:"exclude_files,omitempty"`
}

func (c CmdArchiveExtract) Name() string { return c.Format.extractCmdName() }
func (c CmdArchiveExtract) Validate() error {
	err := c.Format.Validate()
	if err != nil && c.Format != "auto" {
		return err
	}

	return nil

}
func (c CmdArchiveExtract) Resolve() *CommandDefinition {
	return &CommandDefinition{
		CommandName: c.Name(),
		Params:      exportCmd(c),
	}
}

func archiveCreateZipFactory() Command      { return CmdArchiveCreate{Format: ZIP} }
func archiveCreateTarballFactory() Command  { return CmdArchiveCreate{Format: TARBALL} }
func archiveExtractZipFactory() Command     { return CmdArchiveExtract{Format: ZIP} }
func archiveExtractTarballFactory() Command { return CmdArchiveExtract{Format: TARBALL} }
func archiveExtractAutoFactory() Command    { return CmdArchiveExtract{Format: "auto"} }

type CmdAttachArtifacts struct {
	Files    []string `json:"files" yaml:"files"`
	Optional bool     `json:"optional,omitempty" yaml:"optional,omitempty"`
}

func (c CmdAttachArtifacts) Name() string    { return "attach.artifacts" }
func (c CmdAttachArtifacts) Validate() error { return nil }
func (c CmdAttachArtifacts) Resolve() *CommandDefinition {
	return &CommandDefinition{
		CommandName: c.Name(),
		Params:      exportCmd(c),
	}
}
func attachArtifactsFactory() Command { return CmdAttachArtifacts{} }
