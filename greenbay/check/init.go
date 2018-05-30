package check

func init() {
	packageCheckerRegistry = map[string]packageChecker{
		"yum":    packageCheckerFactory([]string{"yum", "list", "installed"}),
		"dpkg":   packageCheckerFactory([]string{"dpkg", "-l"}),
		"brew":   packageCheckerFactory([]string{"brew", "list"}),
		"pacman": packageCheckerFactory([]string{"pacman", "-Q"}),
		"pip":    packageCheckerFactory([]string{"pip", "show"}),
		"gem":    packageCheckerFactory([]string{"gem", "list", "-i"}),
	}

	groupRequirementRegistry = map[string]GroupRequirements{
		"all":  GroupRequirements{All: true},
		"any":  GroupRequirements{Any: true},
		"one":  GroupRequirements{One: true},
		"none": GroupRequirements{None: true},
	}

	registerPackageChecks()       // from package.go
	registerPackageGroupChecks()  // from package_group.go
	registerFileGroupChecks()     // from file_group_exists.go
	registerCommandGroupChecks()  // from command_group.go
	registerSystemLimitChecks()   // from limit.go
	registerProgramChecks()       // from program.go
	registerProgramReturnChecks() // from program_return.go
	registerCompileChecks()       // from compile.go
}
