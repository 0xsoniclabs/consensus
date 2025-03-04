/////////////////////////////////////////////////////////////////////////////
// Lachesis Simulator (Research Prototype)
/////////////////////////////////////////////////////////////////////////////
// (c) 2024 Fantom Foundation / Fantom Research
/////////////////////////////////////////////////////////////////////////////

#include <algorithm>
#include <array>
#include <cassert>
#include <fstream>
#include <functional>
#include <iostream>
#include <map>
#include <set>
#include <sstream>
#include <string>
#include <vector>

#include "lachesis.h"

using namespace std;

/////////////////////////////////////////////////////////////////////////////
// System State
/////////////////////////////////////////////////////////////////////////////


string Lachesis::event_to_string(t_event e) {
  stringstream stream;
  stream << "(" << get<0>(e) << ", " << get<1>(e) << ")";
  return stream.str();
}

bool Lachesis::is_frame_root(t_proc pid, t_event event) {
  for (t_frame frame = 0; frame < (t_frame)frame_roots[pid].size(); frame++) {
    for (const auto &e : frame_roots[pid][frame]) {
      if (e == event) {
        return true;
      }
    }
  }
  return false;
}

bool Lachesis::is_atropos(t_proc pid, t_event event) {
  if (first_atropos == event) {
    return true;
  } else {
    if (atropos_chain.count(event) > 0) {
      return true;
    } else {
      for (const auto &[prev_atropos, current_atropos] : atropos_chain) {
        if (current_atropos == event) {
          return true;
        }
      }
      return false;
    }
  }
}

void Lachesis::dump(t_proc pid, string filename) {
  ofstream os(filename + ".g");
  os << "digraph G {" << endl;

  for (t_proc i = 0; i < num_processors; i++) {
    for (t_seq j = 0; j <= head_seqnum[pid][i]; j++) {
      if (frame_idx[pid][t_event(i, j)] >= 4) {
        os << "node_" << i << "_" << j << " [pos=\"" << i << "," << j
           << "\", label=\"" << i << "," << j << "\"";
        if (is_frame_root(pid, t_event(i, j))) {
          if (is_atropos(pid, t_event(i, j))) {
            os << ", color=green";
          } else {
            os << ", color=red";
          }
        }
        os << "]" << endl;
      }
    }
  }
  for (t_proc i = 0; i < num_processors; i++) {
    for (t_seq j = 0; j <= head_seqnum[pid][i]; j++) {
      for (const t_event &parent : parents[t_event(i, j)]) {
        if (frame_idx[pid][t_event(i, j)] >= 4 && frame_idx[pid][parent] >= 4) {
          os << "node_" << i << "_" << j << "-> node_";
          os << parent.first << "_" << parent.second << endl;
        }
      }
    }
  }
  os << "}" << endl;
}

void Lachesis::dump_state() {
  for (t_proc i = 0; i < num_processors; i++) {
    cout << ";View " << i << endl << "\t";
    for (t_proc j = 0; j < num_processors; j++) {
      cout << head_seqnum[i][j] << " (" << head_seqnum[j][j] << ") ";
    }
    cout << endl;
  }
}

void Lachesis::dump_vectors(string filename) {
  ofstream os(filename + ".txt");

  for (t_proc i = 0; i < num_processors; i++) {
    for (t_seq j = 0; j <= head_seqnum[i][i]; j++) {
      os << "Event (" << i << "," << j << "):" << endl;
      os << "\t downset:";
      for (t_proc k = 0; k < num_processors; k++) {
        if (downset[t_event(i, j)].count(k) > 0) {
          os << "(" << k << "," << downset[t_event(i, j)][k] << ") ";
        }
      }
      os << endl << "\t upset:";
      for (t_proc k = 0; k < num_processors; k++) {
        if (upset[t_event(i, j)].count(k) > 0) {
          os << "(" << k << "," << upset[t_event(i, j)][k] << ") ";
        }
      }
      os << endl;
    }
  }
}

/////////////////////////////////////////////////////////////////////////////
// Assertions
/////////////////////////////////////////////////////////////////////////////

void Lachesis::check_procid(t_proc id) {
  assert(id >= 0 && id < num_processors && "Proc-id is not in range");
}

void Lachesis::check_event(t_event a) {

  // check processor id
  check_procid(a.first);

  // check sequence number
  assert(a.second >= 0 && a.second <= head_seqnum[a.first][a.first] &&
         "Wrong sequence id for event a");

  // check connections to self-parent of event a
  if (a.second > 0) {
    t_event self_parent(a.first, a.second - 1);
    assert(find(parents[a].begin(), parents[a].end(), self_parent) !=
               parents[a].end() &&
           "Self-parent missing");
  }
}
bool Lachesis::check_subsequent_atropos(t_event prev_atropos,
                                        t_event current_atropos) {
  // if a processor has already computed more than one atropos events,
  // this function is checked.

  // check whether the atropos chain has already the current atropos event
  if (atropos_chain.count(prev_atropos) > 0) {
    // we found the current atropos event in the chain
    // which was calculated by another processor.
    if (atropos_chain[prev_atropos] != current_atropos) {
      // bail out: local atropos calculation does not conform with previous
      // atropos calculation of another processor.
      cout << ";Expected atropos: "
           << event_to_string(atropos_chain[prev_atropos]) << endl;
      return false;
    }
  } else {
    // new atropos found in the network; let's update atropos chain
    atropos_chain[prev_atropos] = current_atropos;
  }
  return true;
}

bool Lachesis::check_first_atropos(t_event atropos) {
  if (first_atropos != nil_event) {
    if (first_atropos != atropos) {
      // bail out: local atropos calculation does not conform with previous
      // atropos calculations of other processors.
      return false;
    }
  } else {
    first_atropos = atropos;
  }
  return true;
}

void Lachesis::check_atropos(t_proc pid, t_event atropos) {
  bool correct;
  if (head_atropos.count(pid) == 0) {
    correct = check_first_atropos(atropos);
  } else {
    correct = check_subsequent_atropos(head_atropos[pid], atropos);
  }
  if (!correct) {
    dump_state();
    cout << ";Consensus is inconsistent for processor " << pid << " and event ("
         << atropos.first << "," << atropos.second << ")" << endl;
    exit(1);
  }
  head_atropos[pid] = atropos;
}

void Lachesis::check_frame(t_frame frame, t_event new_event) {
  // check that roots are consistent among processors
  for (t_proc i = 0; i < num_processors; i++) {
    if (frame < (t_frame)frame_roots[i].size()) {
      for (t_event root : frame_roots[i][frame]) {
        if (new_event.first == root.first && new_event.second != root.second) {
          cout << "; New root selection "
               << "(" << new_event.first << "," << new_event.second << ")";
          cout << " of frame " << frame << " diverges from processor " << i
               << " (and may others)."
               << " They have already selected root (" << root.first << ","
               << root.second << ")" << endl;
          dump(new_event.first, "failure.dot");
          exit(1);
        }
      }
    }
  }
}

/////////////////////////////////////////////////////////////////////////////
// Block Generation
/////////////////////////////////////////////////////////////////////////////

void Lachesis::update_upset(t_event new_event, t_event parent_event) {
  t_proc new_pid = new_event.first;
  t_seq new_snum = new_event.second;
  if (upset[parent_event].count(new_pid) == 0) {
    upset[parent_event][new_pid] = new_snum;
    for (auto grandparent : parents[parent_event]) {
      update_upset(new_event, grandparent);
    }
  }
}

t_eventvector Lachesis::join_downset(const t_eventvector &a,
                                     const t_eventvector &b) {
  t_eventvector c;
  for (const auto &[pid, seq] : a) {
    auto it = b.find(pid);
    if (it != b.end()) {
      c[pid] = max(it->second, seq);
    } else {
      c[pid] = seq;
    }
  }
  for (const auto &[pid, seq] : b) {
    if (a.count(pid) == 0) {
      c[pid] = seq;
    }
  }
  return c;
}

bool Lachesis::forkless_cause(t_event a, t_event b) {
  // check correctness of events
  check_event(a);
  check_event(b);

  // count the intersecting space between
  // the upset of b and the downset of a
  uint64_t seen_stake = 0;
  for (const auto &[pid, snum] : upset[b]) {

    // check if pid exists in map
    if (downset[a].count(pid) > 0) {
      // compare sequence numbers
      if (snum <= downset[a][pid]) {
        seen_stake += stake[pid];
      }
    }
  }

  // counter must be greater than or equal to quorum
  return seen_stake >= quorum;
}

/////////////////////////////////////////////////////////////////////////////
// Consensus
/////////////////////////////////////////////////////////////////////////////

void Lachesis::update_frame_atropos(t_proc pid, t_event new_event) {
  // update frame state and check whether new event is a root event. Select legacy calculation if reuqired
  bool is_frame_updated = is_legacy_frame_calc ? update_frame_legacy(pid, new_event) : update_frame(pid, new_event);

  if (is_frame_updated) {
    // update atropos information
    update_atropos(pid, new_event);
  }
}

void Lachesis::choose_atropos(t_proc pid) {
  t_frame frame = last_decided_frame[pid] + 1;
  for (int i = 0; i < num_processors; i++) {
    t_proc j = sorted_pid[i];
    // check if processor has been decided
    if (root_decision[pid][frame].count(j) > 0) {
      // is it a elgible candidate
      if (root_decision[pid][frame][j]) {
        // select atropos
        const auto &it = find_if(
            frame_roots[pid][frame].begin(), frame_roots[pid][frame].end(),
            [&](t_event atropos) { return (atropos.first == j); });
        assert(it != frame_roots[pid][frame].end() &&
               "Atropos decided but not found in frame");

        t_event atropos = *it;

        // check whether atropos selection is consistent
        check_atropos(pid, atropos);

        cout << ";Setting atropos " << event_to_string(atropos)
             << " in processor " << pid << endl;

        // clear root_decision and voting data structure of the frame
        // since it is no longer needed.
        root_decision[pid].erase(frame);
        votes[pid][frame].clear();

        // advance decided frame view of a processor and exit
        last_decided_frame[pid]++;
        return;
      }
    } else {
      // if a more dominant processor is not decided yet,
      // stop choosing an atropos event until it is found.
      return;
    }
  }
}

void Lachesis::perform_aggregation(t_proc pid, t_event new_root) {
  for (t_frame frame = last_decided_frame[pid] + 1;
       frame < frame_idx[pid][new_root] - 1; frame++) {
    t_frame new_root_frame = frame_idx[pid][new_root];
    assert(new_root_frame > frame && "frame overlap error");
    assert(new_root_frame - frame > 1);
    for (t_proc i = 0; i < num_processors; i++) {
      if (root_decision[pid][frame].count(i) == 0) {
        uint64_t num_yes = 0;
        uint64_t num_no = 0;

        for (const auto &root : frame_roots[pid][new_root_frame - 1]) {
          if (forkless_cause(new_root, root)) { // Problem
            t_proc root_proc = root.first;
            if (votes[pid][frame][root][i]) {
              num_yes += stake[root_proc];
            } else {
              num_no += stake[root_proc];
            }
          }
        }
        votes[pid][frame][new_root][i] = num_yes >= num_no;
        if (num_yes >= quorum || num_no >= quorum) {
          root_decision[pid][frame][i] = num_yes >= num_no;
        }
      }
    }
  }
}

void Lachesis::perform_voting(t_proc pid, t_event new_root) {
  assert(last_decided_frame[pid] < frame_idx[pid][new_root] &&
         "Cannot vote on a decided frame");
  // vote on previous frame of new root event
  check_procid(pid);

  // check validity of previous frame
  t_frame frame = frame_idx[pid][new_root] - 1;
  if (frame >= (t_frame)frame_roots[pid].size() || frame < 0) {
    return;
  }

  // on every root in previous we cast a vote with a true/false outcome
  // depending whether the root can be strongly seen by the new found root.
  for (const auto &root : frame_roots[pid][frame]) {
    t_proc root_proc = root.first;
    if (forkless_cause(new_root, root)) {
      votes[pid][frame][new_root][root_proc] = true;
    } else {
      votes[pid][frame][new_root][root_proc] = false;
    }
  }
}

void Lachesis::update_atropos(t_proc pid, t_event new_root) {
  int round = frame_idx[pid][new_root] - last_decided_frame[pid];
  if (round > 0) {
    perform_voting(pid, new_root);
    perform_aggregation(pid, new_root);
    choose_atropos(pid);
  }
}

bool Lachesis::update_frame_legacy(t_proc pid, t_event new_event) {
  check_event(new_event);
  check_procid(pid);

  if (new_event.second <= 0) {
    frame_idx[pid][new_event] = 0;
    insert_frame_root(pid, 0, new_event);
    return true;
  }

  t_event selfparent_event(new_event.first, new_event.second - 1);
  check_event(selfparent_event);

  t_frame selfparent_frame = frame_idx[pid][selfparent_event];
  t_frame max_bound = selfparent_frame + 100;
  t_frame frame = selfparent_frame;

  while (forkless_cause_on_quorum(pid, frame, new_event) && frame < max_bound) {
    frame++;
  }

  frame_idx[pid][new_event] = frame;

  if (frame > selfparent_frame) {
    insert_frame_root(pid, frame, new_event);
    return true;
  } else {
    assert(frame == selfparent_frame &&
           "Frame of new event must be the same as its parent");
    return false;
  }
}

bool Lachesis::update_frame(t_proc pid, t_event new_event) {
  // check arguments
  check_event(new_event);
  check_procid(pid);

  // If it is a genesis event, assign it frame 0.
  if (new_event.second <= 0) {
    frame_idx[pid][new_event] = 0;
    insert_frame_root(pid, 0, new_event);
    return true;
  }

  t_frame max_frame = get_max_parent_frame(pid, new_event);
  t_frame result_frame = max_frame;

  if (forkless_cause_on_quorum(pid, max_frame, new_event)) {
	result_frame++;
  }

  frame_idx[pid][new_event] = result_frame;

  t_event selfparent_event(new_event.first, new_event.second - 1);
  check_event(selfparent_event);
  t_frame selfparent_frame = frame_idx[pid][selfparent_event];

  if (result_frame != selfparent_frame) {
    insert_frame_root(pid, result_frame, new_event);
    return true;
  } else {
    assert(max_frame == result_frame &&
           "Frame of new event must be the same as its max parent");
    return false;
  }
}

bool Lachesis::forkless_cause_on_quorum(t_proc pid, t_frame frame,
                                        t_event new_event) {
  // accumulate stake of roots that forklessly cause the new event
  uint64_t event_stake = 0;
  if (frame < (t_frame)frame_roots[pid].size()) {
    for (t_event root : frame_roots[pid][frame]) {
      if (forkless_cause(new_event, root)) {
        event_stake += stake[root.first];
      }
    }
    return event_stake >= quorum;
  } else {
    return false;
  }
}

t_frame Lachesis::get_max_parent_frame(t_proc pid, t_event new_event) {
  assert(parents[new_event].size() > 0 && "Non genesis node must have parents");
  t_frame frame = 0;
  for (const t_event &parent : parents[new_event]) {
    frame = max(frame, frame_idx[pid][parent]);
  }
  return frame;
}

void Lachesis::insert_frame_root(t_proc pid, t_frame frame, t_event new_event) {
  if (frame >= (t_frame)frame_roots[pid].size()) {
    assert(frame == (t_frame)frame_roots[pid].size() && "Frame index calculation failed");
    frame_roots[pid].resize(frame + 1);
  }
  cout << ";FR " << pid << " " << frame << " " << new_event.first << " "
       << new_event.second << endl;
  frame_roots[pid][frame].insert(new_event);
  // check root consistency
  check_frame(frame, new_event);
}

/////////////////////////////////////////////////////////////////////////////
// State Transitions
/////////////////////////////////////////////////////////////////////////////

// create a new event in processor "producer"
void Lachesis::create_event(t_proc producer,
                            const vector<t_proc> &parent_processors) {
  check_procid(producer);

  // declare new event
  t_event new_event(producer, head_seqnum[producer][producer] + 1);
  t_eventset parent_set;
  t_eventvector new_downset;

  new_downset[producer] = new_event.second;

  for (const t_proc &pid : parent_processors) {

    assert(head_seqnum[producer][pid] >= 0 &&
           "Event missing of parent processor");
    t_event parent_event(pid, head_seqnum[producer][pid]);
    check_event(parent_event);

    // add parent to parent set
    parent_set.insert(parent_event);

    // update upset
    update_upset(new_event, parent_event);

    new_downset = join_downset(new_downset, downset[parent_event]);
  }
  downset[new_event] = new_downset;
  upset[new_event][producer] = new_event.second;

  // create new event for producer by updating parents, descendants,
  // head_seqnum and frame_idx.
  parents[new_event] = parent_set;

  // increment top event
  head_seqnum[producer][producer]++;

  // output newly created event
  cout << "C " << producer;
  for (const t_proc &pid : parent_processors) {
    cout << " " << pid;
  }
  cout << endl;

  // check newly created event
  check_event(new_event);

  // update roots of producer
  update_frame_atropos(producer, new_event);

  // dump state transition
  // dump(producer, "graph_" + to_string(producer) + "_" + to_string(step));
  // dump_vectors("vector_" + to_string(producer) + "_" + to_string(step));
  step++;
}

// receive the next events from processor "sender" in processor "receiver"
// until a given sequence number in the receiver is observed
void Lachesis::receive_event(t_proc receiver, t_proc sender, t_proc seqnum) {
  // check
  check_procid(receiver);
  check_procid(sender);

  // we don't need an update (nothing to receive for a node itself)
  if (receiver == sender) {
    return;
  }

  // update the state of receiver so that it receives the next event of the
  // sender (only if the sender has a new event)
  while (head_seqnum[receiver][sender] < head_seqnum[sender][sender] &&
         head_seqnum[receiver][sender] < seqnum) {
    receive_event(receiver, sender);
  }

  // check whether seqnum coincides with the requested sequence number
  if (head_seqnum[receiver][sender] != seqnum) {
    cout << "Want event (" << sender << "," << seqnum << ")  in processor "
         << receiver << endl;
    cout << "Sequence number is set to " << head_seqnum[receiver][sender]
         << endl;
    exit(1);
  }
}

// receive the next event from processor "sender" in processor "receiver"
void Lachesis::receive_event(t_proc receiver, t_proc sender) {
  check_procid(receiver);
  check_procid(sender);

  // we don't need an update (nothing to receive for a node itself)
  if (receiver == sender) {
    return;
  }

  // update the state of receiver so that it receives the next event of the
  // sender (only if the sender has a new event)
  if (head_seqnum[receiver][sender] < head_seqnum[sender][sender]) {
    t_event new_event(sender, head_seqnum[receiver][sender] + 1);

    // ensure that all parents of the new event are in the local view of the
    // processor (i.e. simulates the reception of all parent events of the new
    // event)
    for (const auto &[parent_pid, parent_seq] : parents[new_event]) {
      while (head_seqnum[receiver][parent_pid] < parent_seq) {
        receive_event(receiver, parent_pid);
      }
    }

    // create new event for receiver
    head_seqnum[receiver][sender]++;
    check_event(new_event);

    // output new received event
    cout << "R " << receiver << " " << sender << endl;

    // update roots of receiver
    update_frame_atropos(receiver, new_event);

    // dump state transition
    // dump(receiver, "graph_" + to_string(receiver) + "_" + to_string(step));
    // dump_vectors("vector_" + to_string(receiver) + "_" + to_string(step));
    step++;
  }
}

/////////////////////////////////////////////////////////////////////////////
// Initialisation of System State
/////////////////////////////////////////////////////////////////////////////

Lachesis::Lachesis(int n, vector<uint64_t> s, bool legacy)
    : num_processors(n), step(1), first_atropos(nil_event), stake(s), is_legacy_frame_calc(legacy) {
  // print init command
  cout << "N " << num_processors;
  for (t_proc k = 0; k < n; k++) {
    cout << " " << s[k];
  }
  cout << endl;

  // compute total stake
  total_stake = 0;
  for (t_proc k = 0; k < n; k++) {
    total_stake += s[k];
  }

  // compute quorum threshold
  quorum = 2 * total_stake / 3 + 1;

  // resize vectors in system state
  head_seqnum.resize(num_processors);
  frame_idx.resize(num_processors);
  votes.resize(num_processors);
  frame_roots.resize(num_processors);
  root_decision.resize(num_processors);
  votes.resize(num_processors);
  sorted_pid.resize(num_processors);

  // sorted PID
  for (t_proc i = 0; i < num_processors; i++) {
    sorted_pid[i] = i;
  }
  sort(sorted_pid.begin(), sorted_pid.end(),
       [&](const t_proc &a, const t_proc &b) {
         check_procid(a);
         check_procid(b);
         return (stake[a] > stake[b] || (stake[a] == stake[b] && a < b));
       });

  // initialise head_seqnum, frame_idx, roots, parents, and descendants state
  for (t_proc i = 0; i < num_processors; i++) {
    // setting the sequence numbers of the most recent event
    // for all processors to the sequence number of the genesis
    // event.
    for (t_proc j = 0; j < num_processors; j++) {
      head_seqnum[i] = vector<t_seq>(num_processors, -1);
    }
    last_decided_frame[i] = -1;
  }
}
