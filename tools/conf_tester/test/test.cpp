// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.


#define CATCH_CONFIG_MAIN
#include "catch.hpp"
#include "../generator.h"

TEST_CASE("LEGACY_DB_SUCCESS") {
    // Should succeed in legacy.
    try{
        EventDbGenerator generator;
        char *argv[] = {(char*)"conf-tester", (char*)"eventdb", (char*)"resources/test-epoch-25101.db", (char*)"25101", (char*)"legacy"};
        int res = generator.process(5, argv);
        REQUIRE(res == 0);
    }
    catch(std::exception& e){
        REQUIRE(false);
    }
}

TEST_CASE("LEGACY_DB_ERROR") {
    // Fail without legacy.
    try{
        EventDbGenerator generator;
        char *argv[] = {(char*)"simulator", (char*)"eventdb", (char*)"resources/test-epoch-25101.db", (char*)"25101"};
        int res = generator.process(4, argv);
        REQUIRE(res == 1);
    }
    catch(std::exception& e){
        // All good.
    }
}

TEST_CASE("POSITIVE_DB_LEGACY") {
    // Should succeed in legacy.
    try{
        EventDbGenerator generator;
        char *argv[] = {(char*)"conf-tester", (char*)"eventdb", (char*)"resources/test-epoch-26000.db", (char*)"26000", (char*)"legacy"};
        int res = generator.process(5, argv);
        REQUIRE(res == 0);
    }
    catch(std::exception& e){
        REQUIRE(false);
    }
}

TEST_CASE("NEGATIVE_DB_NORMAL") {
    // Should succeed in legacy.
    try{
        EventDbGenerator generator;
        char *argv[] = {(char*)"conf-tester", (char*)"eventdb", (char*)"resources/test-epoch-26000.db", (char*)"26000"};
        int res = generator.process(4, argv);
        REQUIRE(res == 0);
    }
    catch(std::exception& e){
        // All good.
    }
}

TEST_CASE("POSITIVE_DB_NORMAL") {
    // Should succeed in normal.
    try{
        EventDbGenerator generator;
        char *argv[] = {(char*)"conf-tester", (char*)"eventdb", (char*)"resources/test-epoch-76.db", (char*)"76"};
        int res = generator.process(4, argv);
        REQUIRE(res == 0);
    }
    catch(std::exception& e){
        REQUIRE(false);
    }
}
