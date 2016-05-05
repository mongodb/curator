# start project configuration
name := curator
buildDir := build
packages := $(name) operations main
projectPath := github.com/mongodb/$(name)
# end project configuration


# start dependency declarations
lintDeps := github.com/alecthomas/gometalinter
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
lint:
	$(gopath)/bin/gometalinter --deadline=20s $(lintExclusion) ./...
build:$(buildDir)/$(name)
test:$(foreach target,$(packages),$(buildDir)/test.$(target).out)
coverage:$(foreach target,$(packages),$(buildDir)/coverage.$(target).out)
coverage-html:$(foreach target,$(packages),$(buildDir)/coverage.$(target).html)
phony := lint build test coverage coverage-html
# end front-ends


# implementation details for building the binary and creating a
# convienent link in the working directory
$(gopath)/src/$(projectPath):deps
	rm -f $@
	ln -s $(shell pwd) $@
$(name):$(buildDir)/$(name)
	[ -L $@ ] || ln -s $< $@
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
test-deps:$(testDeps) $(deps)
lint-deps:$(lintDeps)
	$(gopath)/bin/gometalinter --install
clean:
	rm -rf $(deps) $(lintDeps) $(testDeps) $(buildDir)/test.* $(buildDir)/coverage.*
phony += deps test-deps lint-deps
# end dependency targets

# configure phony targets
.PHONY:$(phony)
