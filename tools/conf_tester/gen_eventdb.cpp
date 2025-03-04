/////////////////////////////////////////////////////////////////////////////
// Lachesis Simulator (Research Prototype)
/////////////////////////////////////////////////////////////////////////////
// (c) 2024 Fantom Foundation / Fantom Research
/////////////////////////////////////////////////////////////////////////////

#include <cassert>
#include <iostream>
#include <random>
#include <sqlite3.h>
#include <sstream>
#include <utility>

#include "generator.h"
#include "lachesis.h"

using namespace std;

// checkAtroposEvent checks whether the event is an Atropos event
inline bool checkAtroposEvent(sqlite3 *db, int64_t event_id, int &validator_id,
                              int &seq_num) {
  bool result = false;
  sqlite3_stmt *stmt;
  stringstream query;
  query << "SELECT Event.ValidatorId, Event.SequenceNumber FROM Atropos, Event"
           " WHERE Atropos.AtroposId = "
        << event_id << " AND Event.EventId = Atropos.AtroposId";
  if (sqlite3_prepare_v2(db, query.str().c_str(), -1, &stmt, 0) != SQLITE_OK) {
    cerr << "simulator: failed to fetch data: " << sqlite3_errmsg(db) << endl;
    sqlite3_close(db);
    throw std::exception();
  }
  if (sqlite3_step(stmt) == SQLITE_ROW) {
    assert(sqlite3_column_count(stmt) == 2 && "Two columns per row expected");
    assert(sqlite3_column_type(stmt, 0) == SQLITE_INTEGER &&
           "Integer expected as first column type");
    assert(sqlite3_column_type(stmt, 1) == SQLITE_INTEGER &&
           "Integer expected as second column type");
    validator_id = sqlite3_column_int(stmt, 0);
    seq_num = sqlite3_column_int(stmt, 1);
    result = true;
  }
  assert(!sqlite3_finalize(stmt) && "Finalize failed");
  return result;
}

// getValidatorStake retrieves validators from the event-db
inline void getValidatorStake(sqlite3 *db, int epoch,
                              vector<uint64_t> &stake_vector,
                              map<int, int> &proc_map) {
  sqlite3_stmt *stmt;
  stringstream query;
  // retrieve validators and stake of validators for an epoch from table
  query << "SELECT ValidatorId, Weight FROM Validator "
        << "WHERE EpochId = " << epoch << " ORDER BY ValidatorId";
  if (sqlite3_prepare_v2(db, query.str().c_str(), -1, &stmt, 0) != SQLITE_OK) {
    cerr << "simulator: failed to fetch data: " << sqlite3_errmsg(db) << endl;
    sqlite3_close(db);
    throw std::exception();
  }

  // create stake vector and create mapping validator-id -> processor
  // Note that validator id number is not dense, i.e., numbers are not strictly
  // drawn from the interval [1,n] and there can be gaps. Hence, we need
  // a processor map.
  while (sqlite3_step(stmt) != SQLITE_DONE) {
    // check result format
    assert(sqlite3_column_count(stmt) == 2 && "Two columns per row expected");
    assert(sqlite3_column_type(stmt, 0) == SQLITE_INTEGER &&
           "Integer expected as first column type");
    assert(sqlite3_column_type(stmt, 1) == SQLITE_INTEGER &&
           "Integer expected as second column type");

    // retrieve row information (ValidatorId, Weight) from Validator table
    // and update stake_vector and processor map that maps validator ids to
    // processor numbers from [0, n-1].
    int validator = sqlite3_column_int(stmt, 0);
    int validator_idx = stake_vector.size();
    proc_map[validator] = validator_idx;
    uint64_t stake = sqlite3_column_int64(stmt, 1);
    stake_vector.push_back(stake);

    cout << "; validator: " << validator_idx << " (" << validator
         << ") stake: " << stake << endl;
  }
  assert(!sqlite3_finalize(stmt) && "Finalize failed");
}

// parent data structure
struct parent {
  int64_t parent_id;
  int validator_id;
  int sequence_number;
};

// getParents retrieves parents of an event from the event-db
inline void getParents(sqlite3 *db, int64_t event_id, vector<parent> &parents) {
  // retrieve validators and stake of validators for an epoch from table
  // Validator
  sqlite3_stmt *stmt;
  stringstream query;
  query << "SELECT p.ParentId, e.ValidatorId, e.SequenceNumber FROM Parent AS "
           "p, Event AS e "
        << "WHERE p.EventId = " << event_id << " AND p.ParentId = e.EventId";
  if (sqlite3_prepare_v2(db, query.str().c_str(), -1, &stmt, 0) != SQLITE_OK) {
    cerr << "simulator: failed to fetch data: " << sqlite3_errmsg(db) << endl;
    sqlite3_close(db);
    throw std::exception();
  }

  // retrieve the parents' validators and their sequence numbers of an event
  while (sqlite3_step(stmt) != SQLITE_DONE) {
    // check result format
    assert(sqlite3_column_count(stmt) == 3 && "Three columns per row expected");
    assert(sqlite3_column_type(stmt, 0) == SQLITE_INTEGER &&
           "Integer expected as first column type (ParentId)");
    assert(sqlite3_column_type(stmt, 1) == SQLITE_INTEGER &&
           "Integer expected as second column type (ValidatorId)");
    assert(sqlite3_column_type(stmt, 2) == SQLITE_INTEGER &&
           "Integer expected as third column type (SequenceNumber)");

    // retrieve row information (ValidatorId, SequenceNumber)
    parent p = {sqlite3_column_int64(stmt, 0), sqlite3_column_int(stmt, 1),
                sqlite3_column_int(stmt, 2)};
    parents.push_back(p);

    // cout << ";\t parent-id: " << p.parent_id << " validator: " <<
    // p.validator_id
    //      << " sequence number: " << p.sequence_number << endl;
  }
  assert(!sqlite3_finalize(stmt) && "Finalize failed");
}

// getEvents retrieves the events of an epoch from the event-db
inline void getEventList(sqlite3 *db, int epoch, set<int64_t> &events) {
  // clear result
  events.clear();

  // find all events of the specified epoch
  stringstream query;
  sqlite3_stmt *stmt;
  query << "SELECT EventId FROM Event WHERE EpochId = " << epoch
        << " ORDER BY EventId";
  if (sqlite3_prepare_v2(db, query.str().c_str(), -1, &stmt, 0) != SQLITE_OK) {
    cerr << "failed to fetch data: " << sqlite3_errmsg(db) << endl;
    sqlite3_close(db);
    throw std::exception();
  }
  while (sqlite3_step(stmt) != SQLITE_DONE) {
    // check result format
    assert(sqlite3_column_count(stmt) == 1 && "Four columns per row expected");
    assert(sqlite3_column_type(stmt, 0) == SQLITE_INTEGER &&
           "Integer expected as first column type (event_id)");
    // retrieve row information (ValidatorId, SequenceNumber)
    int64_t event_id = sqlite3_column_int64(stmt, 0);
    events.insert(event_id);
  }
  assert(!sqlite3_finalize(stmt) && "Finalize failed");
}

// getEvents retrieves the events of an epoch from the event-db
inline void getEvent(sqlite3 *db, int64_t event_id, string &event_hash,
                     int &frame_id, int &validator_id, int &seq_num) {
  // find all events of the specified epoch and process
  // them in event file order
  stringstream query;
  sqlite3_stmt *stmt;
  query << "SELECT EventHash, FrameId, ValidatorId, SequenceNumber  "
           "FROM Event WHERE EventId = "
        << event_id;
  if (sqlite3_prepare_v2(db, query.str().c_str(), -1, &stmt, 0) != SQLITE_OK) {
    cerr << "failed to fetch data: " << sqlite3_errmsg(db) << endl;
    sqlite3_close(db);
    throw std::exception();
  }
  if (sqlite3_step(stmt) != SQLITE_ROW) {
    cerr << "failed to fetch data: " << sqlite3_errmsg(db) << endl;
    sqlite3_close(db);
    throw std::exception();
  }

  // check result format
  assert(sqlite3_column_count(stmt) == 4 && "Four columns per row expected");
  assert(sqlite3_column_type(stmt, 0) == SQLITE_TEXT &&
         "Integer expected as second column type (event_hash)");
  assert(sqlite3_column_type(stmt, 1) == SQLITE_INTEGER &&
         "Integer expected as second column type (frame_id)");
  assert(sqlite3_column_type(stmt, 2) == SQLITE_INTEGER &&
         "Integer expected as third column type (validator_id)");
  assert(sqlite3_column_type(stmt, 3) == SQLITE_INTEGER &&
         "Integer expected as fourth column type (sequence_number)");

  event_hash = (char *)sqlite3_column_text(stmt, 0);
  frame_id = sqlite3_column_int(stmt, 1);
  validator_id = sqlite3_column_int(stmt, 2);
  seq_num = sqlite3_column_int(stmt, 3);
}

/////////////////////////////////////////////////////////////////////////////
// Instance generator for layered instance generation.
/////////////////////////////////////////////////////////////////////////////

int EventDbGenerator::process(int argc, char *argv[]) {

  if (argc < 4 || argc > 5) {
    cerr << "wrong arguments: simulator eventdb <eventdb> <epoch> [legacy]" 

         << endl;
    return 1;
  }

  // open sqlite3 database
  sqlite3 *db;
  if (sqlite3_open(argv[2], &db)) {
    cerr << argv[0] << ": can't open database: " << sqlite3_errmsg(db) << endl;
    return 1;
  }

  // get instance parameters including validators
  // and their stake, epoch number, etc.
  int epoch = stoi(argv[3]);
  vector<uint64_t> stake_vector;
  map<int, int> proc_map;
  getValidatorStake(db, epoch, stake_vector, proc_map);
  int np = stake_vector.size();
  vector<int> frame_vector(np, 1);

  // create new Lachesis instance
  Lachesis l(stake_vector.size(), stake_vector, argc == 5 ? strcmp(argv[4], "legacy") == 0 : false);

  // get set of event-ids to process
  set<int64_t> unprocessed;
  set<int64_t> processed;
  getEventList(db, epoch, unprocessed);

  // vars for atropos check
  t_event prev_atropos;
  bool first_atropos = true;

  while (!unprocessed.empty()) {
    for (int64_t event_id : unprocessed) {

      // get first event-id in set
      string event_hash;
      int frame_id;
      int validator_id;
      int seq_num;

      // read event
      getEvent(db, event_id, event_hash, frame_id, validator_id, seq_num);

      // convert validator-id to processor-id
      if (proc_map.find(validator_id) ==
          proc_map.end()) { // check whether validator_id can be found in
                            // processor map
        cerr << "\tCannot find validator " << validator_id << endl;
        return 1;
      }
      int producer = proc_map[validator_id];
      assert(producer >= 0 && producer < np &&
             "Producer index is out of range");

      // adjust frame number and sequence number to start counting from 0
      frame_id--;
      seq_num--;

      // print basic info
      cout << "; event: " << event_id << " hash: " << event_hash
           << " frame: " << frame_id << " validator: " << producer
           << " sequence-number:" << seq_num << endl;

      // generate event
      vector<parent> parents;
      getParents(db, event_id, parents);

      // check whether parents have already been processed
      bool missingParent = false;
      for (auto &p : parents) {
        if (processed.count(p.parent_id) == 0) {
          missingParent = true;
        }
      }

      // skip current event if parent is missing
      if (missingParent) {
        cout
            << "; Missing parent(s); skip event and find next processable event"
            << endl;
        continue;
      }

      // We receive parent events lazily for the creation of new events when
      // needed.
      vector<t_proc> parent_processors;
      bool foundSelfParent = false;
      for (auto &p : parents) {
        int parent_proc = proc_map[p.validator_id];
        parent_processors.push_back(parent_proc);
        l.receive_event(producer, parent_proc, p.sequence_number - 1);
        if (parent_proc == producer) {
          foundSelfParent = true;
        }
      }
      // fix Lachesis (self-parent is missing for genesis events)
      if (!foundSelfParent) {
        // no self-parent wiring
      }

      // create new event in processor producer
      l.create_event(producer, parent_processors);

      // check frame number
      t_frame fnum = l.get_frame(producer, make_pair(producer, seq_num));
      if (fnum != frame_id) {
        cout << "Frame number of event (" << producer << "," << seq_num << ")"
             << " is " << fnum << " in algorithm." << endl;
        cout << "Event file expects frame number " << frame_id << endl;
        l.dump(producer, "root_failure");
        throw std::exception();
      }

      // update frame vector and print when new frame appears
      if (frame_vector[producer] != frame_id) {
        frame_vector[producer] = frame_id;
        if (!l.is_frame_root(producer, make_pair(producer, seq_num))) {
          cout << "; Event file classifies event as a frame root in frame "
               << frame_id;
          cout << " (is not a frame root in the algorithm!)" << endl;
          l.dump(producer, "root_failure");
          throw std::exception();
        }
      } else {
        if (l.is_frame_root(producer, make_pair(producer, seq_num))) {
          cout << "; Algorithm classifies event as a frame root" << frame_id;
          cout << " (is not a frame root in the event file!)" << endl;
          l.dump(producer, "root_failure");
          throw std::exception();
        }
      }

      // check
      int atropos_validator;
      int atropos_seqnum;
      if (checkAtroposEvent(db, event_id, atropos_validator, atropos_seqnum)) {
        int atropos_id = proc_map[atropos_validator];
        atropos_seqnum--;
        cout << "; Event file classifies event (" << atropos_id << ","
             << atropos_seqnum << ") as atropos." << endl;
        t_event current_atropos = make_pair(atropos_id, atropos_seqnum);
        if (first_atropos) {
          first_atropos = false;
          if (!l.check_first_atropos(current_atropos)) {
            cout << "; (1) Algorithm fails to classify event as atropos"
                 << endl;
            throw std::exception();
          }
        } else {
          if (!l.check_subsequent_atropos(prev_atropos, current_atropos) &&
              current_atropos.second != 1 && current_atropos.second != 3) {
            cout << "; (2) Algorithm fails to classify event as atropos"
                 << endl;
            throw std::exception();
          }
        }
        prev_atropos = current_atropos;
      }
      // mark event as process
      unprocessed.erase(event_id);
      processed.insert(event_id);
      break;
    }
  }

  // close sqlite3 database
  sqlite3_close(db);

  return 0;
}
