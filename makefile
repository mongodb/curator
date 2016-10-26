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
lintArgs := --tests --deadline=1m --vendor
#   gotype produces false positives because it reads .a files which
#   are rarely up to date.
lintArgs += --disable="gotype" --disable="gas"
lintArgs += --skip="build" --skip="buildscripts"
#   enable and configure additional linters
lintArgs += --enable="go fmt -s" --enable="goimports"
lintArgs += --linter='misspell:misspell ./*.go:PATH:LINE:COL:MESSAGE' --enable=misspell
lintArgs += --line-length=100 --dupl-threshold=100 --cyclo-over=15
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
	@-$(gopath)/bin/gometalinter --install >/dev/null
	$(gopath)/bin/gometalinter $(lintArgs) ./...
lint-deps:$(lintDeps)
build:$(buildDir)/$(name)
build-race:$(buildDir)/$(name).race
test:$(foreach target,$(packages),test-$(target))
race:$(foreach target,$(packages),race-$(target))
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


# distribution targets and implementation
dist:$(buildDir)/dist.tar.gz
$(buildDir)/dist.tar.gz:$(buildDir)/$(name)
	tar -C $(buildDir) -czvf $@ $(name)
# end main build


# convenience targets for runing tests and coverage tasks on a
# specific package.
makeArgs := --no-print-directory
race-%:
	@$(MAKE) $(makeArgs) $(buildDir)/race.$*.out
	@grep -s -q -e "^PASS" $(buildDir)/race.$*.out
test-%:
	@$(MAKE) $(makeArgs) $(buildDir)/test.$*.out
	@grep -s -q -e "^PASS" $(buildDir)/test.$*.out
coverage-%:
	@$(MAKE) $(makeArgs) $(buildDir)/coverage.$*.out
	@grep -s -q -e "^PASS" $(buildDir)/coverage.$*.out
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
nestedVendored += vendor/github.com/tychoish/bond
nestedVendored += vendor/github.com/tychoish/lru
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
	rm -rf vendor/github.com/mongodb/amboy/vendor/github.com/tychoish/grip/
	rm -rf vendor/github.com/mongodb/amboy/vendor/github.com/golang.org/x/net/
	rm -rf vendor/github.com/mongodb/amboy/vendor/golang.org/x/net/
	rm -rf vendor/github.com/tychoish/bond/vendor/github.com/tychoish/
	rm -rf vendor/github.com/tychoish/bond/vendor/github.com/mongodb/amboy/
	rm -rf vendor/github.com/tychoish/bond/vendor/golang.org/x/net/
	rm -rf vendor/github.com/tychoish/lru/vendor/github.com/tychoish/grip/
	rm -rf vendor/github.com/tychoish/lru/vendor/github.com/pkg/errors/
	rm -rf vendor/github.com/tychoish/bond/vendor/github.com/davecgh/go-spew/
	rm -rf vendor/github.com/tychoish/bond/vendor/github.com/pkg/errors/
	rm -rf vendor/github.com/tychoish/bond/vendor/github.com/stretchr
	rm -rf vendor/github.com/tychoish/bond/vendor/github.com/satori/
	rm -rf vendor/github.com/tychoish/bond/vendor/github.com/pmezard/
	find vendor/ -name "*.gif" -o -name "*.gz" -o -name "*.png" -o -name "*.ico" -o -name "*testdata*"| xargs rm -rf
change-go-version:
	rm -rf $(buildDir)/make-vendor $(buildDir)/render-gopath
	@$(MAKE) $(makeArgs) vendor > /dev/null 2>&1
vendor:$(buildDir)/vendor/src
	@$(MAKE) $(makeArgs) -C vendor/github.com/tychoish/grip $@
	@-$(MAKE) $(makeArgs) -k -C vendor/github.com/mongodb/amboy $@
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
testArgs := -v --timeout=20m
#    implementation for package coverage and test running, to produce
#    and save test output.
$(buildDir)/coverage.%.html:$(buildDir)/coverage.%.out
	$(vendorGopath) go tool cover -html=$< -o $@
$(buildDir)/coverage.%.out:$(testRunDeps)
	$(vendorGopath) go test -v -covermode=count -coverprofile=$@ $(projectPath)/$*
	@-[ -f $@ ] && go tool cover -func=$@ | sed 's%$(projectPath)/%%' | column -t
$(buildDir)/coverage.$(name).out:$(testRunDeps)
	$(vendorGopath) go test -covermode=count -coverprofile=$@ $(projectPath)
	@-[ -f $@ ] && go tool cover -func=$@ | sed 's%$(projectPath)/%%' | column -t
$(buildDir)/test.%.out:$(testRunDeps)
	$(vendorGopath) go test $(testArgs) ./$* | tee $@
$(buildDir)/test.$(name).out:$(testRunDeps)
	$(vendorGopath) go test $(testArgs) ./ | tee $@
$(buildDir)/race.%.out:$(testRunDeps)
	$(vendorGopath) go test $(testArgs) -race ./$* | tee $@
$(buildDir)/race.$(name).out:$(testRunDeps)
	$(vendorGopath) go test $(testArgs) -race ./ | tee $@
# end test and coverage artifacts


# clean and other utility targets
clean:
	rm -rf $(lintDeps) $(buildDir)/test.* $(buildDir)/coverage.* $(buildDir)/race.* $(name) $(buildDir)/$(name)
phony += clean
# end dependency targets

# configure phony targets
.PHONY:$(phony)
