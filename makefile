# start project configuration
name := curator
buildDir := build
packages := $(name) operations main
orgPath := github.com/mongodb
projectPath := $(orgPath)/$(name)
# end project configuration


# start dependency declarations
lintDeps := github.com/alecthomas/gometalinter
lintDeps += golang.org/x/tools/cmd/gotype
lintDeps += github.com/golang/lint/golint
lintDeps += github.com/kisielk/errcheck
lintDeps += github.com/mdempsky/unconvert
lintDeps += github.com/mvdan/interfacer/cmd/interfacer
lintDeps += github.com/opennota/check/cmd/aligncheck
lintDeps += github.com/opennota/check/cmd/structcheck
lintDeps += github.com/opennota/check/cmd/varcheck
lintDeps += github.com/walle/lll/cmd/lll
lintDeps += honnef.co/go/simple/cmd/gosimple
lintDeps += honnef.co/go/staticcheck/cmd/staticcheck
testDeps := github.com/stretchr/testify
deps := github.com/tychoish/grip
deps += github.com/codegangsta/cli
deps += github.com/blang/semver
# end dependency declarations


# start linting configuration
#   the gotype linter has an imperfect compilation simulator and
#   produces the following false postive errors:
lintExclusion := --exclude="error: could not import github.com/mongodb/curator"
lintExclusion += --exclude="error: undeclared name: .+ \(gotype\)"
#   go lint warns on an error in docstring format, erroneously because
#   it doesn't consider the entire package.
lintExclusion += --exclude="warning: package comment should be of the form \"Package curator ...\""
# end linting configuration


# start dependency installation tools
#   implementation details for being able to lazily install dependencies
gopath := $(shell go env GOPATH)
deps := $(addprefix $(gopath)/src/,$(deps))
lintDeps := $(addprefix $(gopath)/src/,$(lintDeps))
testDeps := $(addprefix $(gopath)/src/,$(testDeps))
$(gopath)/src/%:
	@-[ ! -d $(gopath) ] && mkdir -p $(gopath) || true
	go get $(subst $(gopath)/src/,,$@)
# end dependency installation tools


# userfacing targets for basic build/test/lint operations
lint:$(gopath)/src/$(projectPath) $(lintDeps) $(deps)
	$(gopath)/bin/gometalinter --deadline=20s $(lintExclusion) $< | sed 's%$</%%'
build:deps $(buildDir)/$(name)
test:$(foreach target,$(packages),$(buildDir)/test.$(target).out)
coverage:$(foreach target,$(packages),$(buildDir)/coverage.$(target).out)
coverage-html:$(foreach target,$(packages),$(buildDir)/coverage.$(target).html)
phony := lint build test coverage coverage-html
# end front-ends


# implementation details for building the binary and creating a
# convienent link in the working directory
$(gopath)/src/$(orgPath):
	@mkdir -p $@
$(gopath)/src/$(projectPath):$(gopath)/src/$(orgPath)
	@[ -L $@ ] || ln -s $(shell pwd) $@
$(name):$(buildDir)/$(name)
	@[ -L $@ ] || ln -s $< $@
$(buildDir)/$(name):$(gopath)/src/$(projectPath)
	go build -o $@ main/$(name).go
phony += $(buildDir)/$(name)
# end main build


# convenience targets for runing tests and coverage tasks on a
# specific package.
test-%:
	$(MAKE) $(buildDir)/test.$*.out
coverage-%:
	$(MAKE) $(buildDir)/coverage.$*.out
coverage-html-%:
	$(MAKE) $(buildDir)/coverage.$*.html
phony += $(foreach target,$(packages),test-$(target))
phony += $(foreach target,$(packages),coverage-$(target))
phony += $(foreach target,$(packages),coverage-html-$(target))
# end convienence targets


# start test and coverage artifacts
#    implementation for package coverage and test running, to produce
#    and save test output.
$(buildDir)/coverage.%.html:$(buildDir)/coverage.%.out
	go tool cover -html=$< -o $@
$(buildDir)/coverage.%.out:test-deps
	go test -covermode=count -coverprofile=$@ $(projectPath)/$*
	@-[ -f $@ ] && go tool cover -func=$@ | sed 's%$(projectPath)/%%' | column -t
$(buildDir)/coverage.$(name).out:test-deps
	go test -covermode=count -coverprofile=$@ $(projectPath)
	@-[ -f $@ ] && go tool cover -func=$@ | sed 's%$(projectPath)/%%' | column -t
$(buildDir)/test.%.out:test-deps
	go test -v ./$* >| $@; exitCode=$$?; cat $@; [ $$exitCode -eq 0 ]
$(buildDir)/test.$(name).out:test-deps
	go test -v ./ >| $@; exitCode=$$?; cat $@; [ $$exitCode -eq 0 ]
# end test and coverage artifacts


# start dependency installation (phony) targets.
deps:$(deps)
test-deps:$(testDeps) $(deps) $(name) build
lint-deps:$(lintDeps)
clean:
	rm -rf $(name) $(deps) $(lintDeps) $(testDeps) $(buildDir)/test.* $(buildDir)/coverage.*
phony += deps test-deps lint-deps clean
# end dependency targets

# configure phony targets
.PHONY:$(phony)
