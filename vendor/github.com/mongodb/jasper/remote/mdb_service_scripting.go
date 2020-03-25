package remote

import (
	"context"
	"io"

	"github.com/evergreen-ci/mrpc/mongowire"
)

const (
	ScriptingCreateCommand    = "create_scripting"
	ScriptingGetCommand       = "get_scripting"
	ScriptingSetupCommand     = "setup_scripting"
	ScriptingCleanupCommand   = "cleanup_scripting"
	ScriptingRunCommand       = "run_scripting"
	ScriptingRunScriptCommand = "run_script_scripting"
	ScriptingBuildCommand     = "build_scripting"
	ScriptingTestCommand      = "test_scripting"
)

func (s *mdbService) scriptingGet(ctx context.Context, w io.Writer, msg mongowire.Message)       {}
func (s *mdbService) scriptingCreate(ctx context.Context, w io.Writer, msg mongowire.Message)    {}
func (s *mdbService) scriptingSetup(ctx context.Context, w io.Writer, msg mongowire.Message)     {}
func (s *mdbService) scriptingCleanup(ctx context.Context, w io.Writer, msg mongowire.Message)   {}
func (s *mdbService) scriptingRun(ctx context.Context, w io.Writer, msg mongowire.Message)       {}
func (s *mdbService) scriptingRunScript(ctx context.Context, w io.Writer, msg mongowire.Message) {}
func (s *mdbService) scriptingBuild(ctx context.Context, w io.Writer, msg mongowire.Message)     {}
func (s *mdbService) scriptingTest(ctx context.Context, w io.Writer, msg mongowire.Message)      {}
