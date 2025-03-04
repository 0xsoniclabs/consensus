# Makefile for MAC OS X only
.PHONY: all clean

all: conf-tester

clean: 
	rm -f conf-tester driver.o lachesis.o gen_input.o gen_eventdb.o

conf-tester: driver.o lachesis.o gen_input.o gen_eventdb.o
	g++ -Wall -O3 --std=c++17  driver.o lachesis.o gen_input.o gen_eventdb.o -o conf-tester -lsqlite3

driver.o: driver.cpp generator.h
	g++ -Wall -O3  --std=c++17  driver.cpp -c

lachesis.o: lachesis.cpp lachesis.h
	g++ -Wall -O3  --std=c++17  lachesis.cpp -c

gen_input.o: gen_input.cpp generator.h lachesis.h
	g++ -Wall -O3  --std=c++17  gen_input.cpp -c

gen_eventdb.o: gen_eventdb.cpp generator.h lachesis.h
	g++ -Wall -O3  --std=c++17  gen_eventdb.cpp -c
