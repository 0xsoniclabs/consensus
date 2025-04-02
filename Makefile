# Copyright (c) 2025 Fantom Foundation
#
# Use of this software is governed by the Business Source License included
# in the LICENSE file and at fantom.foundation/bsl11.
#
# Change Date: 2028-4-16
#
# On the date above, in accordance with the Business Source License, use of
# this software will be governed by the GNU Lesser General Public License v3.
#

CXX = g++
FLAGS = -Wall -O3 --std=c++17
CORE_SRC = ./cmd/conf_tester/gen_eventdb.cpp ./cmd/conf_tester/lachesis.cpp ./cmd/conf_tester/gen_input.cpp ./cmd/conf_tester/driver.cpp
TEST_SRC = ./cmd/conf_tester/test/test.cpp ./cmd/conf_tester/gen_eventdb.cpp ./cmd/conf_tester/lachesis.cpp ./cmd/conf_tester/gen_input.cpp
CORE_HPP =./cmd/conf_tester/generator.h ./cmd/conf_tester/lachesis.h 
TEST_HPP = ./cmd/conf_tester/test/catch.hpp ./cmd/conf_tester/generator.h ./cmd/conf_tester/lachesis.h 
CORE_TARGET = ./cmd/conf_tester/conf_tester
TEST_TARGET = ./cmd/conf_tester/test/test

all: conf_tester dbchecker conf_tester_tests

conf_tester: $(CORE_SRC) $(CORE_HPP)
	$(CXX) $(FLAGS) -o $(CORE_TARGET) $(CORE_SRC) -lsqlite3

conf_tester_tests: $(TEST_SRC) $(TEST_HPP)
	$(CXX) $(FLAGS) -o $(TEST_TARGET) $(TEST_SRC) -lsqlite3

dbchecker:
	go build -ldflags="-s -w" -o build/dbchecker ./cmd/dbchecker

.PHONY : test
test :
	go test -shuffle=on ./...
	$(TEST_TARGET)

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
	rm -f $(CORE_TARGET) 
	rm -f ./cmd/conf_tester/*.o
	rm  -f $(TEST_TARGET)
	rm -f ./cmd/conf_tester/test/*.g
	
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
