buildDir := build
srcFiles := $(shell find . -name "*.go" -not -path "./$(buildDir)/*" -not -name "*_test.go" -not -path "*\#*")
testFiles := $(shell find . -name "*.go" -not -path "./$(buildDir)/*" -not -path "*\#*")

_testPackages := ./ ./rpc ./rpc/internal

testArgs := -v
ifneq (,$(RUN_TEST))
testArgs += -run='$(RUN_TEST)'
endif
ifneq (,$(RUN_COUNT))
testArgs += -count='$(RUN_COUNT)'
endif
ifneq (,$(SKIP_LONG))
testArgs += -short
endif

benchPattern := ./

compile:
	go build $(_testPackages)
race:
	@mkdir -p $(buildDir)
	go test $(testArgs) -race $(_testPackages) | tee $(buildDir)/race.poplar.out
	@grep -s -q -e "^PASS" $(buildDir)/race.poplar.out && ! grep -s -q "^WARNING: DATA RACE" $(buildDir)/race.poplar.out
test:
	@mkdir -p $(buildDir)
	go test $(testArgs) $(if $(DISABLE_COVERAGE),, -cover) $(_testPackages) | tee $(buildDir)/test.poplar.out
	@grep -s -q -e "^PASS" $(buildDir)/test.poplar.out
.PHONY: benchmark
benchmark:
	@mkdir -p $(buildDir)
	go test $(testArgs) -bench=$(benchPattern) $(if $(RUN_TEST),, -run=^^$$) | tee $(buildDir)/bench.poplar.out
coverage:$(buildDir)/cover.out
	@go tool cover -func=$< | sed -E 's%github.com/.*/jasper/%%' | column -t
coverage-html:$(buildDir)/cover.html

$(buildDir):$(srcFiles) compile
	@mkdir -p $@
$(buildDir)/cover.out:$(buildDir) $(testFiles) .FORCE
	go test $(testArgs) -coverprofile $@ -cover $(_testPackages)
$(buildDir)/cover.html:$(buildDir)/cover.out
	go tool cover -html=$< -o $@
.FORCE:


proto:vendor/cedar.proto
	@mkdir -p rpc/internal
	protoc --go_out=plugins=grpc:rpc/internal *.proto
	protoc --go_out=plugins=grpc:rpc/internal vendor/cedar.proto
	mv rpc/internal/vendor/cedar.pb.go rpc/internal/cedar.pb.go
clean:
	rm -rf rpc/internal/*.pb.go

vendor/cedar.proto:
	curl -L https://raw.githubusercontent.com/evergreen-ci/cedar/master/perf.proto -o $@
vendor:
	glide install -s


.PHONY:vendor
vendor-clean:
	rm -rf vendor/github.com/mongodb/grip/vendor/github.com/montanaflynn/
	rm -rf vendor/github.com/mongodb/grip/vendor/github.com/stretchr/testify/
	rm -rf vendor/github.com/mongodb/grip/vendor/golang.org/x/sys/
	rm -rf vendor/github.com/mongodb/grip/vendor/github.com/pkg/errors/
	rm -rf vendor/github.com/mongodb/ftdc/vendor/github.com/pkg/errors/
	rm -rf vendor/github.com/mongodb/ftdc/vendor/github.com/mongodb/grip/
	rm -rf vendor/github.com/mongodb/ftdc/vendor/github.com/stretchr/testify
	rm -rf vendor/github.com/mongodb/ftdc/vendor/github.com/mongodb/mongo-go-driver/
	rm -rf vendor/github.com/mongodb/grip/buildscripts/
	rm -rf vendor/github.com/mongodb/mongo-go-driver/vendor/golang.org/x/text/
	rm -rf vendor/github.com/mongodb/mongo-go-driver/vendor/golang.org/x/net/
	rm -rf vendor/github.com/mongodb/mongo-go-driver/vendor/github.com/montanaflynn/
	rm -rf vendor/github.com/mongodb/mongo-go-driver/vendor/github.com/stretchr/
	rm -rf vendor/github.com/mongodb/mongo-go-driver/vendor/github.com/google/go-cmp/cmp/
	rm -rf vendor/github.com/evergreen-ci/pail/vendor/github.com/mongodb/grip/
	rm -rf vendor/github.com/evergreen-ci/pail/vendor/github.com/pkg/errors/
	rm -rf vendor/github.com/evergreen-ci/pail/vendor/github.com/stretchr/testify/
	find vendor/ -name "*.gif" -o -name "*.gz" -o -name "*.png" -o -name "*.ico" -o -name "*.dat" -o -name "*testdata" | xargs rm -rf
