# Copyright (c) 2025 Fantom Foundation
#
# Use of this software is governed by the Business Source License included
# in the LICENSE file and at fantom.foundation/bsl11.
#
# Change Date: 2028-4-16
#
# On the date above, in accordance with the Business Source License, use of
# this software will be governed by the GNU Lesser General Public License v3.

CXX = g++
FLAGS = -Wall -O3 --std=c++17
SRC = test.cpp ../gen_eventdb.cpp ../lachesis.cpp ../gen_input.cpp
HPP = catch.hpp ../generator.h ../lachesis.h 
TARGET = test


all: $(SRC) $(HPP)
	$(CXX) $(FLAGS) -o $(TARGET) $(SRC) -lsqlite3

# Clean up build files
clean:
	rm -f $(TARGET)
