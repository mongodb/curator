# start project configuration
name := curator
buildDir := build
packages := $(name) operations main sthree repobuilder
orgPath := github.com/mongodb
projectPath := $(orgPath)/$(name)
# end project configuration


# start linting configuration
#   package, testing, and linter dependencies specified
#   separately. This is a temporary solution: eventually we should
#   vendorize all of these dependencies.
lintDeps := github.com/alecthomas/gometalinter
#   include test files and give linters 40s to run to avoid timeouts
lintArgs := --tests --deadline=3m --vendor
#   gotype produces false positives because it reads .a files which
#   are rarely up to date.
lintArgs += --disable="gotype" --disable="gas"
lintArgs += --skip="build" --skip="buildscripts"
#   enable and configure additional linters
lintArgs += --enable="goimports"
lintArgs += --line-length=100 --dupl-threshold=150 --cyclo-over=15
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
srcFiles := makefile $(shell find . -name "*.go" -not -path "./$(buildDir)/*" -not -name "*_test.go" -not -path "./buildscripts/*" )
testSrcFiles := makefile $(shell find . -name "*.go" -not -path "./$(buildDir)/*")
testOutput := $(foreach target,$(packages),$(buildDir)/output.$(target).test)
raceOutput := $(foreach target,$(packages),$(buildDir)/output.$(target).race)
coverageOutput := $(foreach target,$(packages),$(buildDir)/output.$(target).coverage)
coverageHtmlOutput := $(foreach target,$(packages),$(buildDir)/output.$(target).coverage.html)
$(gopath)/src/%:
	@-[ ! -d $(gopath) ] && mkdir -p $(gopath) || true
	go get $(subst $(gopath)/src/,,$@)
$(buildDir)/run-linter:buildscripts/run-linter.go $(buildDir)/.lintSetup
	$(vendorGopath) go build -o $@ $<
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
	$(vendorGopath) go build -o $@ main/$(name).go
$(buildDir)/$(name).race:$(srcFiles)
	$(vendorGopath) go build -race -o $@ main/$(name).go
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
#    begin with configuration of dependencies
vendorDeps := github.com/Masterminds/glide
vendorDeps := $(addprefix $(gopath)/src/,$(vendorDeps))
vendor-deps:$(vendorDeps)
#   this allows us to store our vendored code in vendor and use
#   symlinks to support vendored code both in the legacy style and with
#   new-style vendor directories. When this codebase can drop support
#   for go1.4, we can delete most of this.
-include $(buildDir)/makefile.vendor
nestedVendored := vendor/github.com/mongodb/grip
nestedVendored += vendor/github.com/mongodb/amboy
nestedVendored += vendor/github.com/tychoish/bond
nestedVendored += vendor/github.com/tychoish/lru
nestedVendored := $(foreach project,$(nestedVendored),$(project)/build/vendor)
$(buildDir)/makefile.vendor:$(buildDir)/render-gopath makefile
	@mkdir -p $(buildDir)
	@echo "vendorGopath := \$$(shell \$$(buildDir)/render-gopath $(nestedVendored))" >| $@
#   targets for the directory components and manipulating vendored files.
vendor-sync:$(vendorDeps)
	glide install -s
vendor-clean:
	-git checkout vendor/github.com/tychoish/bond/makefile
	rm -rf vendor/github.com/stretchr/testify/vendor/
	rm -rf vendor/github.com/mongodb/amboy/vendor/github.com/mongodb/grip/
	rm -rf vendor/github.com/mongodb/amboy/vendor/github.com/golang.org/x/net/
	rm -rf vendor/github.com/mongodb/amboy/vendor/golang.org/x/net/
	rm -rf vendor/github.com/mongodb/grip/vendor/github.com/stretchr
	rm -rf vendor/github.com/mongodb/grip/vendor/github.com/davecgh
	rm -rf vendor/github.com/mongodb/grip/vendor/github.com/pmezard/go-difflib/
	rm -rf vendor/github.com/tychoish/lru/vendor/github.com/mongodb/grip/
	rm -rf vendor/github.com/tychoish/lru/vendor/github.com/pkg/errors/
	rm -rf vendor/github.com/tychoish/bond/vendor/github.com/tychoish/
	rm -rf vendor/github.com/tychoish/bond/vendor/github.com/mongodb/grip/
	rm -rf vendor/github.com/tychoish/bond/vendor/github.com/mongodb/amboy/
	rm -rf vendor/github.com/tychoish/bond/vendor/golang.org/x/net/
	rm -rf vendor/github.com/tychoish/bond/vendor/github.com/pkg/errors/
	rm -rf vendor/github.com/tychoish/bond/vendor/github.com/davecgh/go-spew/
	rm -rf vendor/github.com/tychoish/bond/vendor/github.com/stretchr
	rm -rf vendor/github.com/tychoish/bond/vendor/github.com/satori/
	rm -rf vendor/github.com/tychoish/bond/vendor/github.com/pmezard/
	rm -rf vendor/github.com/mongodb/amboy/vendor/golang.org/x/net/
	find vendor/ -name "*.gif" -o -name "*.gz" -o -name "*.png" -o -name "*.ico" -o -name "*testdata*"| xargs rm -rf
change-go-version:
	rm -rf $(buildDir)/make-vendor $(buildDir)/render-gopath
	@$(MAKE) $(makeArgs) vendor > /dev/null 2>&1
vendor:$(buildDir)/vendor/src
	@$(MAKE) $(makeArgs) -C vendor/github.com/tychoish/bond $@
	@$(MAKE) $(makeArgs) -C vendor/github.com/mongodb/grip $@
	@$(MAKE) $(makeArgs) -C vendor/github.com/mongodb/amboy $@
$(buildDir)/vendor/src:$(buildDir)/make-vendor $(buildDir)/render-gopath
	@./$(buildDir)/make-vendor
#   targets to build the small programs used to support vendoring.
$(buildDir)/make-vendor:buildscripts/make-vendor.go
	@mkdir -p $(buildDir)
	go build -o $@ $<
$(buildDir)/render-gopath:buildscripts/render-gopath.go
	@mkdir -p $(buildDir)
	go build -o $@ $<
#   define dependencies for buildscripts
buildscripts/make-vendor.go:buildscripts/vendoring/vendoring.go
buildscripts/render-gopath.go:buildscripts/vendoring/vendoring.go
#   add phony targets
phony += vendor vendor-deps vendor-clean vendor-sync change-go-version
# end vendoring tooling configuration


# start test and coverage artifacts
#    tests have compile and runtime deps. This varable has everything
#    that the tests actually need to run. (The "build" target is
#    intentional and makes these targets rerun as expected.)
testArgs := -test.v --test.timeout=15m
ifneq (,$(RUN_TEST))
testArgs += -test.run='$(RUN_TEST)'
endif
#    to avoid vendoring the coverage tool, install it as needed
coverDeps := golang.org/x/tools/cmd/cover
coverDeps := $(addprefix $(gopath)/src/,$(coverDeps))
#    implementation for package coverage and test running,mongodb to produce
#    and save test output.
$(buildDir)/test.operations:$(name)
$(buildDir)/test.%:$(testSrcFiles) $(coverDeps)
	$(vendorGopath) go test $(if $(DISABLE_COVERAGE),,-covermode=count) -c -o $@ ./$(subst -,/,$*)
$(buildDir)/race.operations:$(name)
$(buildDir)/race.%:$(testSrcFiles)
	$(vendorGopath) go test -race -c -o $@ ./$(subst -,/,$*)
#  targets to run any tests in the top-level package
$(buildDir)/test.$(name):$(testSrcFiles) $(coverDeps) $(name)
	$(vendorGopath) go test $(if $(DISABLE_COVERAGE),,-covermode=count) -c -o $@ ./
$(buildDir)/race.$(name):$(testSrcFiles) $(name)
	$(vendorGopath) go test -race -c -o $@ ./
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
	$(vendorGopath) go tool cover -html=$< -o $@
# end test and coverage artifacts


# clean and other utility targets
clean:
	rm -rf $(lintDeps) $(buildDir)/test.* $(buildDir)/coverage.* $(buildDir)/race.* $(name) $(buildDir)/$(name)
phony += clean
# end dependency targets

# configure phony targets
.FORCE:
.PHONY:$(phony) .FORCE
