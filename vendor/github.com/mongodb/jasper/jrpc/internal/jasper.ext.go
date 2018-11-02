package internal

import (
	"syscall"
	"time"

	"github.com/mongodb/grip/send"
	"github.com/mongodb/jasper"
	"github.com/tychoish/bond"
)

func (opts *CreateOptions) Export() *jasper.CreateOptions {
	out := &jasper.CreateOptions{
		Args:             opts.Args,
		Environment:      opts.Environment,
		WorkingDirectory: opts.WorkingDirectory,
		Timeout:          time.Duration(opts.TimeoutSeconds) * time.Second,
		TimeoutSecs:      int(opts.TimeoutSeconds),
		OverrideEnviron:  opts.OverrideEnviron,
		Tags:             opts.Tags,
	}

	if opts.Output != nil {
		out.Output = opts.Output.Export()
	}

	for _, opt := range opts.OnSuccess {
		out.OnSuccess = append(out.OnSuccess, opt.Export())
	}
	for _, opt := range opts.OnFailure {
		out.OnFailure = append(out.OnFailure, opt.Export())
	}
	for _, opt := range opts.OnTimeout {
		out.OnTimeout = append(out.OnTimeout, opt.Export())
	}

	return out
}

func ConvertCreateOptions(opts *jasper.CreateOptions) *CreateOptions {
	if opts.TimeoutSecs == 0 && opts.Timeout != 0 {
		opts.TimeoutSecs = int(opts.Timeout.Seconds())
	}
	output := ConvertOutputOptions(opts.Output)

	co := &CreateOptions{
		Args:             opts.Args,
		Environment:      opts.Environment,
		WorkingDirectory: opts.WorkingDirectory,
		TimeoutSeconds:   int64(opts.TimeoutSecs),
		OverrideEnviron:  opts.OverrideEnviron,
		Tags:             opts.Tags,
		Output:           &output,
	}

	for _, opt := range opts.OnSuccess {
		co.OnSuccess = append(co.OnSuccess, ConvertCreateOptions(opt))
	}
	for _, opt := range opts.OnFailure {
		co.OnFailure = append(co.OnFailure, ConvertCreateOptions(opt))
	}
	for _, opt := range opts.OnTimeout {
		co.OnTimeout = append(co.OnTimeout, ConvertCreateOptions(opt))
	}

	return co
}

func (info *ProcessInfo) Export() jasper.ProcessInfo {
	return jasper.ProcessInfo{
		ID:         info.Id,
		PID:        int(info.Pid),
		IsRunning:  info.Running,
		Successful: info.Successful,
		Complete:   info.Complete,
		Timeout:    info.Timedout,
		Options:    *info.Options.Export(),
	}
}

func ConvertProcessInfo(info jasper.ProcessInfo) *ProcessInfo {
	return &ProcessInfo{
		Id:         info.ID,
		Pid:        int64(info.PID),
		Running:    info.IsRunning,
		Successful: info.Successful,
		Complete:   info.Complete,
		Timedout:   info.Timeout,
		Options:    ConvertCreateOptions(&info.Options),
	}
}

func (s Signals) Export() syscall.Signal {
	switch s {
	case Signals_HANGUP:
		return syscall.SIGHUP
	case Signals_INIT:
		return syscall.SIGINT
	case Signals_TERMINATE:
		return syscall.SIGTERM
	case Signals_KILL:
		return syscall.SIGKILL
	default:
		return syscall.Signal(0)
	}
}

func ConvertSignal(s syscall.Signal) Signals {
	switch s {
	case syscall.SIGHUP:
		return Signals_HANGUP
	case syscall.SIGINT:
		return Signals_INIT
	case syscall.SIGTERM:
		return Signals_TERMINATE
	case syscall.SIGKILL:
		return Signals_KILL
	default:
		return Signals_UNKNOWN
	}
}

func ConvertFilter(f jasper.Filter) *Filter {
	switch f {
	case jasper.All:
		return &Filter{Name: FilterSpecifications_ALL}
	case jasper.Running:
		return &Filter{Name: FilterSpecifications_RUNNING}
	case jasper.Terminated:
		return &Filter{Name: FilterSpecifications_TERMINATED}
	case jasper.Failed:
		return &Filter{Name: FilterSpecifications_FAILED}
	case jasper.Successful:
		return &Filter{Name: FilterSpecifications_SUCCESSFUL}
	default:
		return nil
	}
}

func ConvertLogType(lt jasper.LogType) LogType {
	switch lt {
	case jasper.LogBuildloggerV2:
		return LogType_LOGBUILDLOGGERV2
	case jasper.LogBuildloggerV3:
		return LogType_LOGBUILDLOGGERV3
	case jasper.LogDefault:
		return LogType_LOGDEFAULT
	case jasper.LogFile:
		return LogType_LOGFILE
	case jasper.LogInherit:
		return LogType_LOGINHERIT
	case jasper.LogSplunk:
		return LogType_LOGSPLUNK
	case jasper.LogSumologic:
		return LogType_LOGSUMOLOGIC
	case jasper.LogInMemory:
		return LogType_LOGINMEMORY
	default:
		return LogType_LOGUNKNOWN
	}
}

func (lt LogType) Export() jasper.LogType {
	switch lt {
	case LogType_LOGBUILDLOGGERV2:
		return jasper.LogBuildloggerV2
	case LogType_LOGBUILDLOGGERV3:
		return jasper.LogBuildloggerV3
	case LogType_LOGDEFAULT:
		return jasper.LogDefault
	case LogType_LOGFILE:
		return jasper.LogFile
	case LogType_LOGINHERIT:
		return jasper.LogInherit
	case LogType_LOGSPLUNK:
		return jasper.LogSplunk
	case LogType_LOGSUMOLOGIC:
		return jasper.LogSumologic
	case LogType_LOGINMEMORY:
		return jasper.LogInMemory
	default:
		return jasper.LogType("")
	}
}

func ConvertOutputOptions(opts jasper.OutputOptions) OutputOptions {
	loggers := []*Logger{}
	for _, logger := range opts.Loggers {
		loggers = append(loggers, ConvertLogger(logger))
	}
	return OutputOptions{
		SuppressOutput:        opts.SuppressOutput,
		SuppressError:         opts.SuppressError,
		RedirectOutputToError: opts.SendOutputToError,
		RedirectErrorToOutput: opts.SendErrorToOutput,
		Loggers:               loggers,
	}
}

func ConvertLogger(logger jasper.Logger) *Logger {
	return &Logger{
		LogType:    ConvertLogType(logger.Type),
		LogOptions: ConvertLogOptions(logger.Options),
	}
}

func ConvertLogOptions(opts jasper.LogOptions) *LogOptions {
	return &LogOptions{
		BufferOptions:      ConvertBufferOptions(opts.BufferOptions),
		BuildloggerOptions: ConvertBuildloggerOptions(opts.BuildloggerOptions),
		DefaultPrefix:      opts.DefaultPrefix,
		FileName:           opts.FileName,
		Format:             ConvertLogFormat(opts.Format),
		InMemoryCap:        int64(opts.InMemoryCap),
		SplunkOptions:      ConvertSplunkOptions(opts.SplunkOptions),
		SumoEndpoint:       opts.SumoEndpoint,
	}
}

func ConvertBufferOptions(opts jasper.BufferOptions) *BufferOptions {
	return &BufferOptions{
		Buffered: opts.Buffered,
		Duration: int64(opts.Duration),
		MaxSize:  int64(opts.MaxSize),
	}
}

func ConvertBuildloggerOptions(opts send.BuildloggerConfig) *BuildloggerOptions {
	return &BuildloggerOptions{
		CreateTest: opts.CreateTest,
		Url:        opts.URL,
		Number:     int64(opts.Number),
		Phase:      opts.Phase,
		Builder:    opts.Builder,
		Test:       opts.Test,
		Command:    opts.Command,
	}
}

func ConvertBuildloggerURLs(urls []string) *BuildloggerURLs {
	u := &BuildloggerURLs{Urls: []string{}}
	for _, url := range urls {
		u.Urls = append(u.Urls, url)
	}
	return u
}

func (u *BuildloggerURLs) Export() []string {
	urls := []string{}
	for _, url := range u.Urls {
		urls = append(urls, url)
	}
	return urls
}

func ConvertSplunkOptions(opts send.SplunkConnectionInfo) *SplunkOptions {
	return &SplunkOptions{
		Url:     opts.ServerURL,
		Token:   opts.Token,
		Channel: opts.Channel,
	}
}

func ConvertLogFormat(f jasper.LogFormat) LogFormat {
	switch f {
	case jasper.LogFormatDefault:
		return LogFormat_LOGFORMATDEFAULT
	case jasper.LogFormatJSON:
		return LogFormat_LOGFORMATJSON
	case jasper.LogFormatPlain:
		return LogFormat_LOGFORMATPLAIN
	default:
		return LogFormat_LOGFORMATUNKNOWN
	}
}

func (f LogFormat) Export() jasper.LogFormat {
	switch f {
	case LogFormat_LOGFORMATDEFAULT:
		return jasper.LogFormatDefault
	case LogFormat_LOGFORMATJSON:
		return jasper.LogFormatJSON
	case LogFormat_LOGFORMATPLAIN:
		return jasper.LogFormatPlain
	default:
		return jasper.LogFormatInvalid
	}
}

func (opts OutputOptions) Export() jasper.OutputOptions {
	loggers := []jasper.Logger{}
	for _, logger := range opts.Loggers {
		loggers = append(loggers, logger.Export())
	}
	return jasper.OutputOptions{
		SuppressOutput:    opts.SuppressOutput,
		SuppressError:     opts.SuppressError,
		SendOutputToError: opts.RedirectOutputToError,
		SendErrorToOutput: opts.RedirectErrorToOutput,
		Loggers:           loggers,
	}
}

func (logger Logger) Export() jasper.Logger {
	return jasper.Logger{
		Type:    logger.LogType.Export(),
		Options: logger.LogOptions.Export(),
	}
}

func (opts LogOptions) Export() jasper.LogOptions {
	out := jasper.LogOptions{
		DefaultPrefix: opts.DefaultPrefix,
		FileName:      opts.FileName,
		Format:        opts.Format.Export(),
		InMemoryCap:   int(opts.InMemoryCap),
		SumoEndpoint:  opts.SumoEndpoint,
	}

	if opts.SplunkOptions != nil {
		out.SplunkOptions = opts.SplunkOptions.Export()
	}
	if opts.BufferOptions != nil {
		out.BufferOptions = opts.BufferOptions.Export()
	}
	if opts.BuildloggerOptions != nil {
		out.BuildloggerOptions = opts.BuildloggerOptions.Export()
	}

	return out
}

func (opts *BufferOptions) Export() jasper.BufferOptions {
	return jasper.BufferOptions{
		Buffered: opts.Buffered,
		Duration: time.Duration(opts.Duration),
		MaxSize:  int(opts.MaxSize),
	}
}

func (opts BuildloggerOptions) Export() send.BuildloggerConfig {
	return send.BuildloggerConfig{
		CreateTest: opts.CreateTest,
		URL:        opts.Url,
		Number:     int(opts.Number),
		Phase:      opts.Phase,
		Builder:    opts.Builder,
		Test:       opts.Test,
		Command:    opts.Command,
	}
}

func (opts SplunkOptions) Export() send.SplunkConnectionInfo {
	return send.SplunkConnectionInfo{
		ServerURL: opts.Url,
		Token:     opts.Token,
		Channel:   opts.Channel,
	}
}

func (opts *BuildOptions) Export() bond.BuildOptions {
	return bond.BuildOptions{
		Target:  opts.Target,
		Arch:    bond.MongoDBArch(opts.Arch),
		Edition: bond.MongoDBEdition(opts.Edition),
		Debug:   opts.Debug,
	}
}

func (opts *MongoDBDownloadOptions) Export() jasper.MongoDBDownloadOptions {
	jopts := jasper.MongoDBDownloadOptions{
		BuildOpts: opts.BuildOptions.Export(),
		Path:      opts.Path,
	}

	jopts.Releases = make([]string, 0, len(opts.Releases))
	for _, release := range opts.Releases {
		jopts.Releases = append(jopts.Releases, release)
	}
	return jopts
}

func (opts *CacheOptions) Export() jasper.CacheOptions {
	return jasper.CacheOptions{
		Disabled:   opts.Disabled,
		PruneDelay: time.Duration(opts.PruneDelay),
		MaxSize:    int(opts.MaxSize),
	}
}

func ConvertDownloadInfo(info jasper.DownloadInfo) *DownloadInfo {
	return &DownloadInfo{
		Path:        info.Path,
		Url:         info.URL,
		ArchiveOpts: ConvertArchiveOptions(info.ArchiveOpts),
	}
}

func (info *DownloadInfo) Export() jasper.DownloadInfo {
	return jasper.DownloadInfo{
		Path:        info.Path,
		URL:         info.Url,
		ArchiveOpts: info.ArchiveOpts.Export(),
	}
}

func ConvertArchiveFormat(format jasper.ArchiveFormat) ArchiveFormat {
	switch format {
	case jasper.ArchiveAuto:
		return ArchiveFormat_ARCHIVEAUTO
	case jasper.ArchiveTarGz:
		return ArchiveFormat_ARCHIVETARGZ
	case jasper.ArchiveZip:
		return ArchiveFormat_ARCHIVEZIP
	default:
		return ArchiveFormat_ARCHIVEUNKNOWN
	}
}

func (format ArchiveFormat) Export() jasper.ArchiveFormat {
	switch format {
	case ArchiveFormat_ARCHIVEAUTO:
		return jasper.ArchiveAuto
	case ArchiveFormat_ARCHIVETARGZ:
		return jasper.ArchiveTarGz
	case ArchiveFormat_ARCHIVEZIP:
		return jasper.ArchiveZip
	default:
		return jasper.ArchiveFormat("")
	}
}

func ConvertArchiveOptions(opts jasper.ArchiveOptions) *ArchiveOptions {
	return &ArchiveOptions{
		ShouldExtract: opts.ShouldExtract,
		Format:        ConvertArchiveFormat(opts.Format),
		TargetPath:    opts.TargetPath,
	}
}

func (opts ArchiveOptions) Export() jasper.ArchiveOptions {
	return jasper.ArchiveOptions{
		ShouldExtract: opts.ShouldExtract,
		Format:        opts.Format.Export(),
		TargetPath:    opts.TargetPath,
	}
}
