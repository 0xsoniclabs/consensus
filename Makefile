# Copyright (c) 2025 Fantom Foundation
#
# Use of this software is governed by the Business Source License included
# in the LICENSE file and at fantom.foundation/bsl11.
#
# Change Date: 2028-4-16
#
# On the date above, in accordance with the Business Source License, use of
# this software will be governed by the GNU Lesser General Public License v3.

dbchecker:
	go build -ldflags="-s -w" -o build/dbchecker ./cmd/dbchecker

.PHONY : test
test :
	go test -shuffle=on ./...

.PHONY : test-race
test-race :
	go test -shuffle=on -race -timeout=20m ./...

.PHONY: coverage
coverage:
	go test -count=1 -shuffle=on -covermode=atomic -coverpkg=./... -coverprofile=cover.prof ./...
	go tool cover -func cover.prof | grep -e "^total:"

.PHONY : clean
clean :
	rm -fr ./build/*

.PHONY : lint
lint:
	@./build/bin/golangci-lint run --config ./.golangci.yml

.PHONY : lintci-deps
lintci-deps:
	rm -f ./build/bin/golangci-lint
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ./build/bin v1.52.2

.PHONY : install-deps
install-deps:
	go get github.com/JekaMas/go-mutesting/cmd/go-mutesting@v1.1.2

.PHONY : mut
mut:
	MUTATION_TEST=on go-mutesting --blacklist=".github/mut_blacklist" --config=".github/mut_config.yml" ./... &> .stats.msi
	@echo MSI: `jq '.stats.msi' report.json`
