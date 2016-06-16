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
#   explicitly download linters used by the metalinter so that we can
#   avoid using the installation/update options, which often hit
#   timeouts and do not propagate dependency installation errors.
lintDeps += github.com/alecthomas/gocyclo
lintDeps += github.com/golang/lint/golint
lintDeps += github.com/gordonklaus/ineffassign
lintDeps += github.com/jgautheron/goconst/cmd/goconst
lintDeps += github.com/kisielk/errcheck
lintDeps += github.com/mdempsky/unconvert
lintDeps += github.com/mibk/dupl
lintDeps += github.com/mvdan/interfacer/cmd/interfacer
lintDeps += github.com/opennota/check/cmd/aligncheck
lintDeps += github.com/opennota/check/cmd/structcheck
lintDeps += github.com/opennota/check/cmd/varcheck
lintDeps += github.com/tsenart/deadcode
lintDeps += github.com/client9/misspell/cmd/misspell
lintDeps += github.com/walle/lll/cmd/lll
lintDeps += honnef.co/go/simple/cmd/gosimple
lintDeps += honnef.co/go/staticcheck/cmd/staticcheck
# end dependency declarations


# start linting configuration
#   include test files and give linters 40s to run to avoid timeouts
lintArgs := --tests --deadline=40s
#   skip the build directory and the gopath,
lintArgs += --skip="$(gopath)" --skip="$(buildDir)" --skip "vendor"
#   gotype produces false positives because it reads .a files which
#   are rarely up to date.
lintArgs += --disable="gotype"
#   enable and configure additional linters
lintArgs += --enable="go fmt -s" --enable="goimports"
lintArgs += --linter='misspell:misspell ./*.go:PATH:LINE:COL:MESSAGE' --enable=misspell
lintArgs += --line-length=100 --dupl-threshold=100
#   the gotype linter has an imperfect compilation simulator and
#   produces the following false postive errors:
lintArgs += --exclude="error: could not import github.com/mongodb/curator"
#   go lint warns on an error in docstring format, erroneously because
#   it doesn't consider the entire package.
lintArgs += --exclude="warning: package comment should be of the form \"Package .* ...\""
# end linting configuration


# start dependency installation tools
#   implementation details for being able to lazily install dependencies
gopath := $(shell go env GOPATH)
lintDeps := $(addprefix $(gopath)/src/,$(lintDeps))
srcFiles := makefile $(shell find . -name "*.go" -not -path "./$(buildDir)/*" -not -name "*_test.go" -not -path "./buildscripts/*" )
testSrcFiles := makefile $(shell find . -name "*.go" -not -path "./$(buildDir)/*")
testOutput := $(foreach target,$(packages),$(buildDir)/test.$(target).out)
raceOutput := $(foreach target,$(packages),$(buildDir)/race.$(target).out)
coverageOutput := $(foreach target,$(packages),$(buildDir)/coverage.$(target).out)
coverageHtmlOutput := $(foreach target,$(packages),$(buildDir)/coverage.$(target).html)
$(gopath)/src/%:
	@-[ ! -d $(gopath) ] && mkdir -p $(gopath) || true
	go get $(subst $(gopath)/src/,,$@)
# end dependency installation tools


# userfacing targets for basic build and development operations
lint:$(lintDeps)
	$(gopath)/bin/gometalinter $(lintArgs) ./... | sed 's%$</%%'
lint-deps:$(lintDeps)
build:$(buildDir)/$(name)
build-race:$(buildDir)/$(name).race
test:$(testOutput)
race:$(raceOutput)
coverage:$(coverageOutput)
coverage-html:$(coverageHtmlOutput)
phony := lint lint-deps build build-race race test coverage coverage-html
.PRECIOUS: $(testOutput) $(raceOutput) $(coverageOutput) $(coverageHtmlOutput)
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


# convenience targets for runing tests and coverage tasks on a
# specific package.
makeArgs := --no-print-directory
race-%:
	@$(MAKE) $(makeArgs) $(buildDir)/race.$*.out
test-%:
	@$(MAKE) $(makeArgs) $(buildDir)/test.$*.out
coverage-%:
	@$(MAKE) $(makeArgs) $(buildDir)/coverage.$*.out
html-coverage-%:
	@$(MAKE) $(makeArgs) $(buildDir)/coverage.$*.html
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
nestedVendored := vendor/github.com/tychoish/grip
nestedVendored += vendor/github.com/mongodb/amboy
nestedVendored := $(foreach project,$(nestedVendored),$(project)/build/vendor)
$(buildDir)/makefile.vendor:$(buildDir)/render-gopath makefile
	@mkdir -p $(buildDir)
	@echo "vendorGopath := \$$(shell \$$(buildDir)/render-gopath $(nestedVendored))" >| $@
#   targets for the directory components and manipulating vendored files.
vendor-sync:$(vendorDeps)
	glide install -s
vendor-clean:
	rm -rf vendor/github.com/stretchr/testify/vendor/
	find vendor/ -name "*.gif" -o -name "*.gz" -o -name "*.png" -o -name "*.ico" | xargs rm -f
change-go-version:
	rm -rf $(buildDir)/make-vendor $(buildDir)/render-gopath
	@$(MAKE) $(makeArgs) vendor > /dev/null 2>&1
vendor:$(buildDir)/vendor/src
	$(MAKE) $(makeArgs) -C vendor/github.com/tychoish/grip $@
	$(MAKE) $(makeArgs) -C vendor/github.com/mongodb/amboy $@
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
#    This varable includes everything that the tests actually need to
#    run. (The "build" target is intentional and makes these targetsb
#    rerun as expected.)
testRunDeps := $(testSrcFiles) $(name) build
#    implementation for package coverage and test running, to produce
#    and save test output.
$(buildDir)/coverage.%.html:$(buildDir)/coverage.%.out
	$(vendorGopath) go tool cover -html=$< -o $@
$(buildDir)/coverage.%.out:$(testRunDeps)
	$(vendorGopath) go test -covermode=count -coverprofile=$@ $(projectPath)/$*
	@-[ -f $@ ] && go tool cover -func=$@ | sed 's%$(projectPath)/%%' | column -t
$(buildDir)/coverage.$(name).out:$(testRunDeps)
	$(vendorGopath) go test -covermode=count -coverprofile=$@ $(projectPath)
	@-[ -f $@ ] && go tool cover -func=$@ | sed 's%$(projectPath)/%%' | column -t
$(buildDir)/test.%.out:$(testRunDeps)
	$(vendorGopath) go test -v ./$* >| $@; exitCode=$$?; cat $@; [ $$exitCode -eq 0 ]
$(buildDir)/test.$(name).out:$(testRunDeps)
	$(vendorGopath) go test -v ./ >| $@; exitCode=$$?; cat $@; [ $$exitCode -eq 0 ]
$(buildDir)/race.%.out:$(testRunDeps)
	$(vendorGopath) go test -race -v ./$* >| $@; exitCode=$$?; cat $@; [ $$exitCode -eq 0 ]
$(buildDir)/race.$(name).out:$(testRunDeps)
	$(vendorGopath) go test -race -v ./ >| $@; exitCode=$$?; cat $@; [ $$exitCode -eq 0 ]
# end test and coverage artifacts


# clean and other utility targets
clean:
	rm -rf $(lintDeps) $(buildDir)/test.* $(buildDir)/coverage.* $(buildDir)/race.* $(name) $(buildDir)/$(name)
phony += clean
# end dependency targets

# configure phony targets
.PHONY:$(phony)
