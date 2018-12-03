# start project configuration
name := curator
buildDir := build
packages := $(name) operations cmd-curator sthree repobuilder
packages += greenbay greenbay-check
orgPath := github.com/mongodb
projectPath := $(orgPath)/$(name)
# end project configuration


# start build configuratino
ldFlags := $(if $(DEBUG_ENABLED),,-w -s)
ldFlags += -X=github.com/mongodb/curator.BuildRevision=$(shell git rev-parse HEAD)
ldFlags += -X=github.com/mongodb/curator.JasperChecksum=$(shell shasum vendor/github.com/mongodb/jasper/jasper.proto | cut -f 1 -d ' ')
# end build configuration


# start linting configuration
#   package, testing, and linter dependencies specified
#   separately. This is a temporary solution: eventually we should
#   vendorize all of these dependencies.
lintDeps := github.com/alecthomas/gometalinter
#   include test files and give linters 40s to run to avoid timeouts
lintArgs := --tests --deadline=3m --vendor
#   gotype produces false positives because it reads .a files which
#   are rarely up to date.
lintArgs += --disable="gotype" --disable="gosec"
lintArgs += --skip="build"
#   enable and configure additional linters
lintArgs += --enable="goimports"
lintArgs += --line-length=100 --dupl-threshold=150 --cyclo-over=25
#   the gotype linter has an imperfect compilation simulator and
#   produces the following false postive errors:
lintArgs += --exclude="error: could not import github.com/mongodb/curator"
#   go lint warns on an error in docstring format, erroneously because
#   it doesn't consider the entire package.
lintArgs += --exclude="warning: package comment should be of the form \"Package .* ...\""
#   known issues that the linter picks up that are not relevant in our cases
lintArgs += --exclude="warning: jobImpl is unused.*" # this interface is used for testing
lintArgs += --exclude="file is not goimported" # top-level mains aren't imported
lintArgs += --exclude="duplicate of (rpm|deb)_test.go"  # these suites are very similar by design
lintArgs += --exclude="unused struct field .*repobuilder.Job.mutex" # this is used by type that compose this type.
lintArgs += --exclude="should check returned error before deferring.*Close()"
# end linting configuration


# start dependency installation tools
#   implementation details for being able to lazily install dependencies
.DEFAULT_GOAL := $(name)
gopath := $(shell go env GOPATH)
lintDeps := $(addprefix $(gopath)/src/,$(lintDeps))
srcFiles := makefile $(shell find . -name "*.go" -not -path "./$(buildDir)/*" -not -name "*_test.go" )
testSrcFiles := makefile $(shell find . -name "*.go" -not -path "./$(buildDir)/*")
testOutput := $(foreach target,$(packages),$(buildDir)/output.$(target).test)
raceOutput := $(foreach target,$(packages),$(buildDir)/output.$(target).race)
coverageOutput := $(foreach target,$(packages),$(buildDir)/output.$(target).coverage)
coverageHtmlOutput := $(foreach target,$(packages),$(buildDir)/output.$(target).coverage.html)
$(gopath)/src/%:
	@-[ ! -d $(gopath) ] && mkdir -p $(gopath) || true
	go get $(subst $(gopath)/src/,,$@)
$(buildDir)/run-linter:cmd/run-linter/run-linter.go $(buildDir)/.lintSetup
	go build -o $@ $<
$(buildDir)/.lintSetup:$(lintDeps)
	@-$(gopath)/bin/gometalinter --install >/dev/null && touch $@
# end dependency installation tools


# userfacing targets for basic build and development operations
lint:$(buildDir)/output.lint
build:$(buildDir)/$(name)
build-race:$(buildDir)/$(name).race
test:$(foreach target,$(packages),test-$(target))
race:$(foreach target,$(packages),race-$(target))
coverage:$(coverageOutput)
coverage-html:$(coverageHtmlOutput)
phony := lint build build-race race test coverage coverage-html
.PRECIOUS:$(testOutput) $(raceOutput) $(coverageOutput) $(coverageHtmlOutput)
.PRECIOUS:$(foreach target,$(packages),$(buildDir)/test.$(target))
.PRECIOUS:$(foreach target,$(packages),$(buildDir)/race.$(target))
.PRECIOUS:$(foreach target,$(packages),$(buildDir)/output.$(target).lint)
.PRECIOUS:$(buildDir)/output.lint
# end front-ends


# implementation details for building the binary and creating a
# convienent link in the working directory
$(name):$(buildDir)/$(name)
	@[ -e $@ ] || ln -s $<
$(buildDir)/$(name):$(srcFiles)
	go build -ldflags="$(ldFlags)" -o $@ cmd/$(name)/$(name).go
$(buildDir)/$(name).race:$(srcFiles)
	go build -ldflags="$(ldFlags)" -race -o $@ cmd/$(name)/$(name).go
phony += $(buildDir)/$(name)
# end main build


# distribution targets and implementation
dist:$(buildDir)/dist.tar.gz
$(buildDir)/dist.tar.gz:$(buildDir)/$(name)
	tar -C $(buildDir) -czvf $@ $(name)
# end main build


# convenience targets for runing tests and coverage tasks on a
# specific package.
race-%:$(buildDir)/output.%.race
	@grep -s -q -e "^PASS" $< && ! grep -s -q "^WARNING: DATA RACE" $<
test-%:$(buildDir)/output.%.test
	@grep -s -q -e "^PASS" $<
coverage-%:$(buildDir)/output.%.coverage
	@grep -s -q -e "^PASS" $(subst coverage,test,$<)
html-coverage-%:$(buildDir)/output.%.coverage $(buildDir)/output.%.coverage.html
	@grep -s -q -e "^PASS" $(subst coverage,test,$<)
lint-%:$(buildDir)/output.%.lint
	@grep -v -s -q "^--- FAIL" $<
# end convienence targets


# start vendoring configuration
vendor-clean:
	rm -rf vendor/github.com/evergreen-ci/gimlet/vendor/github.com/mongodb/grip
	rm -rf vendor/github.com/evergreen-ci/gimlet/vendor/github.com/stretchr/testify/
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/github.com/golang/protobuf
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/github.com/mongodb/grip
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/github.com/mongodb/ftdc
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/github.com/stretchr/testify
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/golang.org/x/net
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/golang.org/x/sys
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/google.golang.org/genproto
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/google.golang.org/grpc
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/golang.org/x/text/
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/github.com/pkg/errors/
	rm -rf vendor/github.com/mongodb/amboy/vendor/github.com/mongodb/grip/
	rm -rf vendor/github.com/mongodb/amboy/vendor/github.com/pkg/errors/
	rm -rf vendor/github.com/mongodb/amboy/vendor/github.com/stretchr/testify/
	rm -rf vendor/github.com/mongodb/ftdc/vendor/github.com/mongodb/grip
	rm -rf vendor/github.com/mongodb/ftdc/vendor/github.com/mongodb/mongo-go-driver
	rm -rf vendor/github.com/mongodb/ftdc/vendor/github.com/mongodb/mongo-go-driver/vendor/gopkg.in/yaml.v2/
	rm -rf vendor/github.com/mongodb/ftdc/vendor/github.com/pkg/errors
	rm -rf vendor/github.com/mongodb/ftdc/vendor/github.com/satori/go.uuid
	rm -rf vendor/github.com/mongodb/ftdc/vendor/github.com/stretchr/testify
	rm -rf vendor/github.com/mongodb/ftdc/vendor/github.com/stretchr/testify
	rm -rf vendor/github.com/mongodb/grip/vendor/github.com/stretchr/testify/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/evergreen-ci/gimlet
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/pkg/errors/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/golang/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/golang/protobuf
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/mongodb/amboy
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/mongodb/grip
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/tychoish/bond
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/tychoish/lru
	rm -rf vendor/github.com/mongodb/jasper/vendor/golang.org/x/net
	rm -rf vendor/github.com/mongodb/jasper/vendor/golang.org/x/sys
	rm -rf vendor/github.com/mongodb/jasper/vendor/golang.org/x/text/
	rm -rf vendor/github.com/mongodb/jasper/vendor/google.golang.org/genproto
	rm -rf vendor/github.com/mongodb/jasper/vendor/google.golang.org/grpc
	rm -rf vendor/github.com/papertrail/go-tail/main.go
	rm -rf vendor/github.com/papertrail/go-tail/vendor/github.com/spf13/pflag/
	rm -rf vendor/github.com/papertrail/go-tail/vendor/golang.org/x/sys/
	rm -rf vendor/github.com/tychoish/bond/vendor/github.com/mongodb/amboy/
	rm -rf vendor/github.com/tychoish/bond/vendor/github.com/mongodb/grip/
	rm -rf vendor/github.com/tychoish/bond/vendor/github.com/pkg/errors/
	rm -rf vendor/github.com/tychoish/bond/vendor/github.com/satori/go.uuid/
	rm -rf vendor/github.com/tychoish/bond/vendor/github.com/stretchr/testify/
	rm -rf vendor/github.com/tychoish/lru/vendor/github.com/mongodb/grip/
	rm -rf vendor/github.com/tychoish/lru/vendor/github.com/pkg/errors/
	find vendor/ -name "*.gif" -o -name "*.gz" -o -name "*.png" -o -name "*.ico" -o -name "*testdata*"| xargs rm -rf
#   add phony targets
phony += vendor-clean
# end vendoring tooling configuration


# start test and coverage artifacts
#    tests have compile and runtime deps. This varable has everything
#    that the tests actually need to run. (The "build" target is
#    intentional and makes these targets rerun as expected.)
testArgs := -test.v --test.timeout=15m
ifneq (,$(RUN_TEST))
testArgs += -test.run='$(RUN_TEST)'
endif
ifneq (,$(RUN_CASE))
testArgs += -testify.m='$(RUN_CASE)'
endif
#    to avoid vendoring the coverage tool, install it as needed
coverDeps := $(if $(DISABLE_COVERAGE),,golang.org/x/tools/cmd/cover)
coverDeps := $(addprefix $(gopath)/src/,$(coverDeps))
#    implementation for package coverage and test running,mongodb to produce
#    and save test output.
$(buildDir)/test.operations:$(name)
$(buildDir)/test.%:$(testSrcFiles) $(coverDeps)
	go test $(if $(DISABLE_COVERAGE),,-covermode=count) -c -o $@ ./$(subst -,/,$*)
$(buildDir)/race.operations:$(name)
$(buildDir)/race.%:$(testSrcFiles)
	go test -race -c -o $@ ./$(subst -,/,$*)
#  targets to run any tests in the top-level package
$(buildDir)/test.$(name):$(testSrcFiles) $(coverDeps)
	go test $(if $(DISABLE_COVERAGE),,-covermode=count) -c -o $@ ./
$(buildDir)/race.$(name):$(testSrcFiles)
	go test -race -c -o $@ ./
#  targets to run the tests and report the output
$(buildDir)/output.%.test:$(buildDir)/test.% .FORCE
	$(testRunEnv) ./$< $(testArgs) 2>&1 | tee $@
$(buildDir)/output.%.race:$(buildDir)/race.% .FORCE
	$(testRunEnv) ./$< $(testArgs) 2>&1 | tee $@
#  targets to generate gotest output from the linter.
$(buildDir)/output.%.lint:$(buildDir)/run-linter $(testSrcFiles) .FORCE
	@./$< --output=$@ --lintArgs='$(lintArgs)' --packages='$*'
$(buildDir)/output.lint:$(buildDir)/run-linter .FORCE
	@./$< --output="$@" --lintArgs='$(lintArgs)' --packages="$(packages)"
#  targets to process and generate coverage reports
$(buildDir)/output.%.coverage:$(buildDir)/test.% .FORCE $(coverDeps)
	$(testRunEnv) ./$< $(testArgs) -test.coverprofile=$@ | tee $(subst coverage,test,$@)
	@-[ -f $@ ] && go tool cover -func=$@ | sed 's%$(projectPath)/%%' | column -t
$(buildDir)/output.%.coverage.html:$(buildDir)/output.%.coverage $(coverDeps)
	go tool cover -html=$< -o $@
# end test and coverage artifacts


# clean and other utility targets
clean:
	rm -rf $(lintDeps) $(buildDir)/test.* $(buildDir)/coverage.* $(buildDir)/race.* $(name) $(buildDir)/$(name)
phony += clean
# end dependency targets

# configure phony targets
.FORCE:
.PHONY:$(phony) .FORCE
