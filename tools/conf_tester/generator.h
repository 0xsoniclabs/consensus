// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

#pragma once

// Instance generator interface for running a simulation
class Generator {
public:
  virtual int process(int argc, char *argv[]) = 0;
};

// Read the operations from the console
class InputGenerator : public Generator {
public:
  int process(int argc, char *argv[]);
};

// Construct instance from an event db
class EventDbGenerator : public Generator {
public:
  int process(int argc, char *argv[]);
};
