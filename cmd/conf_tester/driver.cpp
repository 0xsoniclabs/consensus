// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

/////////////////////////////////////////////////////////////////////////////
// Driver that chooses an instance generator based on the first argument
// and creates the Lachesis simulation with the chosen instance generator.
/////////////////////////////////////////////////////////////////////////////

#include <cassert>
#include <iostream>
#include <sstream>
#include <utility>

#include "generator.h"

using namespace std;

// instance generator registry
InputGenerator input_generator;
EventDbGenerator eventdb_generator; 

struct {
  const char *name;
  Generator *generator;
} registry[] = {
                {"input", &input_generator},
                {"eventdb", &eventdb_generator},
	       };

// find an instance generator and call it
int main(int argc, char *argv[]) {
  // check argument format
  if (argc <= 1) {
    cerr << "command missing" << endl;
    return 1;
  }

  // find instance generator
  for (int i = 0; i < end(registry) - begin(registry); i++) {
    if (string(argv[1]) == string(registry[i].name)) {
      return registry[i].generator->process(argc, argv);
    }
  }

  // command was not found
  cerr << argv[0] << ": unknown command " << argv[1] << endl;
  return 1;
}
