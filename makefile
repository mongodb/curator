# start project configuration
name := curator
ifeq (${GOOS}, windows)
	binary := $(name).exe
else
	binary := $(name)
endif
buildDir := build
packages := operations cmd-curator repobuilder greenbay greenbay-check
srcFiles := makefile $(shell find . -name "*.go" -not -path "./$(buildDir)/*" -not -name "*_test.go" )
testSrcFiles := makefile $(shell find . -name "*.go" -not -path "./$(buildDir)/*")
orgPath := github.com/mongodb
projectPath := $(orgPath)/$(name)
# end project configuration


# start environment setup
gobin := go
ifneq (,$(GOROOT))
gobin := $(GOROOT)/bin/go
endif

ifeq ($(OS),Windows_NT)
gobin := $(shell cygpath $(gobin))
export GOCACHE := $(shell cygpath -m $(abspath $(buildDir)/.cache))
export GOLANGCI_LINT_CACHE := $(shell cygpath -m $(abspath $(buildDir)/.lint-cache))
export GOPATH := $(shell cygpath -m $(GOPATH))
export GOROOT := $(shell cygpath -m $(GOROOT))
endif

export GO111MODULE := off
# end environment setup

# Ensure the build directory exists, since most targets require it.
$(shell mkdir -p $(buildDir))

.DEFAULT_GOAL := $(binary)

# start lint setup targets
lintDeps := $(buildDir)/golangci-lint $(buildDir)/run-linter
$(buildDir)/golangci-lint:
	@curl --retry 10 --retry-max-time 60 -sSfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(buildDir) v1.30.0 >/dev/null 2>&1
$(buildDir)/run-linter: cmd/run-linter/run-linter.go $(buildDir)/golangci-lint
	$(gobin) build -o $@ $<
# end lint setup targets

# start output files
lintOutput := $(foreach target,$(packages),$(buildDir)/output.$(target).lint)
testOutput := $(foreach target,$(packages),$(buildDir)/output.$(target).test)
coverageOutput := $(foreach target,$(packages),$(buildDir)/output.$(target).coverage)
coverageHtmlOutput := $(foreach target,$(packages),$(buildDir)/output.$(target).coverage.html)
.PRECIOUS: $(testOutput) $(lintOutput) $(coverageOutput) $(coverageHtmlOutput)
# end output files

# start basic development operations
compile: $(buildDir)/$(binary)
test: $(testOutput)
lint: $(lintOutput)
coverage: $(coverageOutput)
coverage-html: $(coverageHtmlOutput)
revendor: $(buildDir)/$(binary)
	$(buildDir)/$(binary) revendor $(if $(VENDOR_REVISION),--revision $(VENDOR_REVISION),) $(if $(VENDOR_PKG),--package $(VENDOR_PKG) ,) $(if $(VENDOR_CLEAN),--clean "$(MAKE) vendor-clean",)
phony := compile lint test coverage coverage-html

# start convenience targets for running tests and coverage tasks on a
# specific package.
test-%: $(buildDir)/output.%.test
	@grep -s -q -e "^PASS" $<
coverage-%: $(buildDir)/output.%.coverage
	@grep -s -q -e "^PASS" $(subst coverage,test,$<)
html-coverage-%: $(buildDir)/output.%.coverage $(buildDir)/output.%.coverage.html
	@grep -s -q -e "^PASS" $(subst coverage,test,$<)
lint-%: $(buildDir)/output.%.lint
	@grep -v -s -q "^--- FAIL" $<
# end convenience targets
# end basic development operations

# start test and coverage artifacts
testArgs := -v -timeout=15m
ifneq (,$(RUN_TEST))
testArgs += -run='$(RUN_TEST)'
endif
ifneq (,$(RUN_COUNT))
testArgs += -count=$(RUN_COUNT)
endif
ifeq (,$(DISABLE_COVERAGE))
testArgs += -cover
endif
ifneq (,$(RACE_DETECTOR))
testArgs += -race
endif
$(buildDir)/output.%.test: .FORCE
	$(gobin) test $(testArgs) ./$(if $(subst $(name),,$*),$(subst -,/,$*),) 2>&1 | tee $@
$(buildDir)/output.%.coverage: .FORCE
	$(gobin) test $(testArgs) ./$(if $(subst $(name),,$*),$(subst -,/,$*),) -covermode=count -coverprofile $@ | tee $(subst coverage,test,$@)
	@-[ -f $@ ] && $(gobin) tool cover -func=$@ | sed 's%$(projectPath)/%%' | column -t
$(buildDir)/output.%.coverage.html: $(buildDir)/output.%.coverage
	$(gobin) tool cover -html=$< -o $@

ifneq (go,$(gobin))
# We have to handle the PATH specially for linting in CI, because if the PATH has a different version of the Go
# binary in it, the linter won't work properly.
lintEnvVars := PATH="$(shell dirname $(gobin)):$(PATH)"
endif
$(buildDir)/output.%.lint: $(buildDir)/run-linter .FORCE
	@$(lintEnvVars) ./$< --output=$@ --lintBin=$(buildDir)/golangci-lint --packages='$*'
# end test and coverage artifacts

# start cli and distribution targets
ldFlags += $(if $(DEBUG_ENABLED),,-w -s)
ldFlags += -X=github.com/mongodb/curator.BuildRevision=$(shell git rev-parse HEAD)
# convenience link in the working directory to the binary
$(binary): $(buildDir)/$(binary)
	@[ -e $@ ] || ln -s $<
$(buildDir)/$(binary): $(srcFiles)
	$(gobin) build -ldflags="$(ldFlags)" -o $@ cmd/$(name)/$(name).go
phony += $(buildDir)/$(binary)
dist: $(buildDir)/dist.tar.gz
$(buildDir)/dist.tar.gz: $(buildDir)/$(binary)
	tar -C $(buildDir) -czvf $@ $(binary)
# end cli and distribution targets

# start vendoring configuration
vendor-clean:
	rm -rf vendor/github.com/evergreen-ci/gimlet/vendor/github.com/mongodb/grip
	rm -rf vendor/github.com/evergreen-ci/gimlet/vendor/github.com/stretchr/testify/
	rm -rf vendor/github.com/evergreen-ci/gimlet/vendor/go.mongodb.org/
	rm -rf vendor/github.com/evergreen-ci/pail/vendor/github.com/evergreen-ci/poplar/
	rm -rf vendor/github.com/evergreen-ci/pail/vendor/github.com/evergreen-ci/utility/
	rm -rf vendor/github.com/evergreen-ci/pail/vendor/github.com/jmespath/go-jmespath/
	rm -rf vendor/github.com/evergreen-ci/pail/vendor/github.com/mongodb/grip/
	rm -rf vendor/github.com/evergreen-ci/pail/vendor/github.com/pkg/errors/
	rm -rf vendor/github.com/evergreen-ci/pail/vendor/github.com/stretchr/
	rm -rf vendor/github.com/evergreen-ci/pail/vendor/go.mongodb.org/mongo-driver/
	rm -rf vendor/github.com/evergreen-ci/pail/{scripts,cmd,evergreen.yaml}
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/github.com/golang/protobuf
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/github.com/google/
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/github.com/evergreen-ci/aviation/
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/github.com/evergreen-ci/birch/
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/github.com/evergreen-ci/pail/
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/github.com/evergreen-ci/birch/
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/github.com/evergreen-ci/juniper/
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/github.com/mongodb/amboy/
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/github.com/mongodb/grip/
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/github.com/mongodb/ftdc/
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/github.com/papertrail/go-tail/
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/github.com/pkg/errors/
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/github.com/stretchr/testify/
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/golang.org/x/net/
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/golang.org/x/sys/
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/google.golang.org/genproto/
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/google.golang.org/grpc/
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/golang.org/x/text/
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/go.mongodb.org/mongo-driver/
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/gopkg.in/yaml.v2/
	rm -rf vendor/github.com/mongodb/amboy/vendor/github.com/aws/aws-sdk-go/
	rm -rf vendor/github.com/mongodb/amboy/vendor/github.com/google/uuid/
	rm -rf vendor/github.com/mongodb/amboy/vendor/github.com/urfave/cli/
	rm -rf vendor/github.com/mongodb/amboy/vendor/go.mongodb.org/
	rm -rf vendor/github.com/mongodb/amboy/vendor/github.com/google/uuid
	rm -rf vendor/github.com/mongodb/amboy/vendor/github.com/mongodb/grip/
	rm -rf vendor/github.com/mongodb/amboy/vendor/github.com/pkg/errors/
	rm -rf vendor/github.com/mongodb/amboy/vendor/github.com/stretchr/testify/
	rm -rf vendor/github.com/mongodb/anser/vendor/go.mongodb.org/mongo-driver/
	rm -rf vendor/github.com/mongodb/anser/vendor/github.com/stretchr/
	rm -rf vendor/github.com/mongodb/anser/vendor/github.com/pkg/errors/
	rm -rf vendor/github.com/mongodb/anser/vendor/github.com/mongodb/grip/
	rm -rf vendor/github.com/mongodb/anser/vendor/github.com/mongodb/amboy/
	rm -rf vendor/github.com/mongodb/anser/vendor/github.com/mongodb/ftdc/
	rm -rf vendor/github.com/mongodb/anser/vendor/github.com/satori/go.uuid/
	rm -rf vendor/github.com/mongodb/anser/vendor/github.com/evergreen-ci/birch
	rm -rf vendor/github.com/mongodb/ftdc/vendor/go.mongodb.org/
	rm -rf vendor/github.com/mongodb/ftdc/vendor/github.com/papertrail/
	rm -rf vendor/github.com/mongodb/ftdc/vendor/github.com/evergreen-ci/birch/
	rm -rf vendor/github.com/mongodb/ftdc/vendor/github.com/mongodb/grip
	rm -rf vendor/github.com/mongodb/ftdc/vendor/github.com/evergreen-ci/birch
	rm -rf vendor/github.com/mongodb/ftdc/vendor/github.com/pkg/errors
	rm -rf vendor/github.com/mongodb/ftdc/vendor/github.com/satori/go.uuid
	rm -rf vendor/github.com/mongodb/ftdc/vendor/github.com/stretchr/testify
	rm -rf vendor/github.com/mongodb/ftdc/vendor/github.com/stretchr/testify
	rm -rf vendor/github.com/mongodb/ftdc/vendor/go.mongodb.org/mongo-driver/
	rm -rf vendor/github.com/mongodb/grip/vendor/github.com/google/uuid/
	rm -rf vendor/github.com/mongodb/grip/vendor/github.com/pkg/
	rm -rf vendor/github.com/mongodb/grip/vendor/github.com/stretchr/testify/
	rm -rf vendor/github.com/mongodb/grip/vendor/github.com/google/uuid
	# TODO (EVG-14482): riscv64 isn't recognized as a valid GOARCH until go1.16, so we can likely stop removing this
	# file once CI tests are compiled with go1.16+.
	rm -rf vendor/github.com/mongodb/grip/vendor/github.com/tklauser/go-sysconf/zsysconf_values_linux_riscv64.go
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/docker/docker/vendor/github.com/aws/aws-sdk-go/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/evergreen-ci/aviation/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/evergreen-ci/birch/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/evergreen-ci/gimlet/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/evergreen-ci/poplar/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/evergreen-ci/pail/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/evergreen-ci/timber/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/evergreen-ci/utility/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/golang/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/google/uuid/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/mongodb/amboy/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/mongodb/ftdc/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/mongodb/grip/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/pkg/errors/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/evergreen-ci/bond/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/evergreen-ci/lru
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/PuerkitoBio/rehttp/
	rm -rf vendor/github.com/mongodb/jasper/vendor/golang.org/x/net/
	rm -rf vendor/github.com/mongodb/jasper/vendor/golang.org/x/sys/
	rm -rf vendor/github.com/mongodb/jasper/vendor/golang.org/x/text/
	rm -rf vendor/github.com/mongodb/jasper/vendor/golang.org/x/oauth2/
	rm -rf vendor/github.com/mongodb/jasper/vendor/gopkg.in/yaml.v2/
	rm -rf vendor/github.com/mongodb/jasper/vendor/google.golang.org/genproto/
	rm -rf vendor/github.com/mongodb/jasper/vendor/google.golang.org/grpc/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/stretchr/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/urfave/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/jpillora/backoff/
	rm -rf vendor/github.com/mongodb/jasper/vendor/go.mongodb.org/mongo-driver/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/sabhiram/go-git-ignore/
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/github.com/PuerkitoBio/rehttp/
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/github.com/evergreen-ci/utility
	rm -rf vendor/github.com/evergreen-ci/poplar/vendor/golang.org/x/oauth2/
	rm -rf vendor/go.mongodb.org/mongo-driver/data/
	rm -rf vendor/go.mongodb.org/mongo-driver/vendor/github.com/davecgh
	rm -rf vendor/go.mongodb.org/mongo-driver/vendor/github.com/montanaflynn
	rm -rf vendor/go.mongodb.org/mongo-driver/vendor/github.com/pmezard
	rm -rf vendor/go.mongodb.org/mongo-driver/vendor/github.com/stretchr
	rm -rf vendor/go.mongodb.org/mongo-driver/vendor/golang.org/x/net
	rm -rf vendor/go.mongodb.org/mongo-driver/vendor/golang.org/x/text
	rm -rf vendor/github.com/papertrail/go-tail/main.go
	rm -rf vendor/github.com/papertrail/go-tail/vendor/github.com/spf13/pflag/
	rm -rf vendor/github.com/papertrail/go-tail/vendor/golang.org/x/sys/
	rm -rf vendor/github.com/evergreen-ci/bond/vendor/github.com/mongodb/amboy/
	rm -rf vendor/github.com/evergreen-ci/bond/vendor/github.com/mongodb/grip/
	rm -rf vendor/github.com/evergreen-ci/bond/vendor/github.com/pkg/errors/
	rm -rf vendor/github.com/evergreen-ci/bond/vendor/github.com/satori/go.uuid/
	rm -rf vendor/github.com/evergreen-ci/bond/vendor/github.com/stretchr/testify/
	rm -rf vendor/github.com/evergreen-ci/bond/vendor/github.com/google/uuid/
	rm -rf vendor/github.com/evergreen-ci/lru/vendor/github.com/mongodb/grip/
	rm -rf vendor/github.com/evergreen-ci/lru/vendor/github.com/pkg/errors/
	rm -rf vendor/github.com/evergreen-ci/aviation/vendor/github.com/evergreen-ci/gimlet/
	rm -rf vendor/github.com/evergreen-ci/aviation/vendor/github.com/jpillora/backoff/
	rm -rf vendor/github.com/evergreen-ci/aviation/vendor/github.com/mongodb/grip/
	rm -rf vendor/github.com/evergreen-ci/aviation/vendor/github.com/pkg/errors/
	rm -rf vendor/github.com/evergreen-ci/aviation/vendor/github.com/stretchr/testify/
	rm -rf vendor/github.com/evergreen-ci/aviation/vendor/google.golang.org/genproto/
	rm -rf vendor/github.com/evergreen-ci/aviation/vendor/google.golang.org/grpc/
	rm -rf vendor/github.com/evergreen-ci/timber/vendor/go.mongodb.org/
	rm -rf vendor/github.com/evergreen-ci/timber/vendor/github.com/evergreen-ci/aviation/
	rm -rf vendor/github.com/evergreen-ci/timber/vendor/github.com/evergreen-ci/utility/
	rm -rf vendor/github.com/evergreen-ci/timber/vendor/github.com/evergreen-ci/juniper/
	rm -rf vendor/github.com/evergreen-ci/timber/vendor/github.com/golang/protobuf/
	rm -rf vendor/github.com/evergreen-ci/timber/vendor/github.com/mongodb/grip/
	rm -rf vendor/github.com/evergreen-ci/timber/vendor/github.com/stretchr/testify/
	rm -rf vendor/github.com/evergreen-ci/timber/vendor/github.com/PuerkitoBio/rehttp/
	rm -rf vendor/github.com/evergreen-ci/timber/vendor/golang.org/x/net/
	rm -rf vendor/github.com/evergreen-ci/timber/vendor/golang.org/x/sys/
	rm -rf vendor/github.com/evergreen-ci/timber/vendor/golang.org/x/text/
	rm -rf vendor/github.com/evergreen-ci/timber/vendor/google.golang.org/genproto/
	rm -rf vendor/github.com/evergreen-ci/timber/vendor/google.golang.org/grpc/
	rm -rf vendor/github.com/evergreen-ci/timber/vendor/github.com/pkg/errors/
	rm -rf vendor/github.com/evergreen-ci/timber/vendor/gopkg.in/yaml.v2/
	rm -rf vendor/github.com/evergreen-ci/timber/internal/formats.pb.go
	rm -rf vendor/github.com/evergreen-ci/timber/internal/system_metrics.pb.go
	rm -rf vendor/github.com/mholt/archiver/tarsz.go
	rm -rf vendor/github.com/mholt/archiver/rar.go
	rm -rf vendor/github.com/mholt/archiver/tarlz4.go
	rm -rf vendor/github.com/mholt/archiver/tarbz2.go
	rm -rf vendor/github.com/mholt/archiver/tarxz.go
	find vendor/ -name "*.gif" -o -name "*.gz" -o -name "*.png" -o -name "*.ico" -o -name "*testdata*"| xargs rm -rf
phony += vendor-clean
# end vendoring configuration

# start cleanup targets
clean:
	rm -rf $(buildDir) $(binary)
clean-results:
	rm -rf $(buildDir)/output.*
phony += clean clean-results
# end cleanup targets

# configure phony targets
.FORCE:
.PHONY: $(phony) .FORCE
