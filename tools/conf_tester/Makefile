# Copyright (c) 2025 Fantom Foundation
#
# Use of this software is governed by the Business Source License included
# in the LICENSE file and at fantom.foundation/bsl11.
#
# Change Date: 2028-4-16
#
# On the date above, in accordance with the Business Source License, use of
# this software will be governed by the GNU Lesser General Public License v3.


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
