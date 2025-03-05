/////////////////////////////////////////////////////////////////////////////
// Consensus Simulator (Research Prototype)
/////////////////////////////////////////////////////////////////////////////
// (c) 2024 Sonic Labs / Sonic Research
/////////////////////////////////////////////////////////////////////////////

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
