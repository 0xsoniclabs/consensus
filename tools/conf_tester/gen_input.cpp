/////////////////////////////////////////////////////////////////////////////
// Consensus Simulator (Research Prototype)
/////////////////////////////////////////////////////////////////////////////
// (c) 2024 Sonic Labs / Sonic Research
/////////////////////////////////////////////////////////////////////////////
 

#include <cassert>
#include <iostream>
#include <memory>
#include <random>
#include <sstream>
#include <utility>

#include "generator.h"
#include "lachesis.h"

using namespace std;

/////////////////////////////////////////////////////////////////////////////
// Instance generator that reads operations from console
/////////////////////////////////////////////////////////////////////////////

int InputGenerator::process(int argc, char *argv[]) {
  unique_ptr<Lachesis> l;
  if (argc < 2 || argc > 3) {
    cerr << "wrong arguments: simulator input [legacy]" << endl;
    return 1;
  }
  string line;
  while (getline(cin, line)) {
    stringstream ls(line);
    string command;
    ls >> command;
    if (command == "N") {
      int n;
      ls >> n;
      vector<uint64_t> stake_vector;
      for (int i = 0; i < n; i++) {
        uint64_t stake;
        ls >> stake;
        stake_vector.push_back(stake);
      }
      l = make_unique<Lachesis>(n, stake_vector, argc == 3 ? strcmp(argv[2], "legacy") == 0 : false);
    } else if (command == "C") {
      t_proc producer, parent_pid;
      ls >> producer;
      vector<t_proc> parent_processors;
      while (ls >> parent_pid) {
        parent_processors.push_back(parent_pid);
      }
      l->create_event(producer, parent_processors);
    } else if (command == "R") {
      t_proc receiver, sender;
      ls >> receiver;
      ls >> sender;
      l->receive_event(receiver, sender);
    } else if (command[0] != ';') {
      cerr << "Unknown command" << endl;
      exit(1);
    }
  }
  return 0;
}
