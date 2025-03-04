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

