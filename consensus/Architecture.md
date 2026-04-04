# Sonic Consensus Architecture

Sonic Consensus is a DAG-based asynchronous Byzantine Fault Tolerant (BFT) consensus library implementing the Lachesis algorithm. It achieves finality without timeouts, tolerates up to 1/3 Byzantine validators, and detects equivocations (double-signing) through branch tracking.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          Host Application                               │
│                                                                         │
│  EventSource          ConsensusCallbacks                                │
│  ┌──────────────┐     ┌──────────────────────────────────────────┐      │
│  │ HasEvent()   │     │ BeginBlock(Block)                        │      │
│  │ GetEvent()   │     │   ApplyEvent(Event)                      │      │
│  └──────┬───────┘     │   EndBlock() -> *Validators (epoch seal) │      │
│         │             └───────────────────┬──────────────────────┘      │
└─────────┼─────────────────────────────────┼─────────────────────────────┘
          │                                 │
          │  provides events                │  receives decisions
          │                                 │
┌─────────┼─────────────────────────────────┼─────────────────────────────┐
│         ▼                                 ▼                             │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                      consensusengine                            │    │
│  │                                                                 │    │
│  │  IndexedLachesis                                                │    │
│  │  ┌───────────────────────────────────────────────────────┐      │    │
│  │  │  Lachesis                                             │      │    │
│  │  │  ┌─────────────────────────────────────────────────┐  │      │    │
│  │  │  │  Orderer                                        │  │      │    │
│  │  │  │                                                 │  │      │    │
│  │  │  │  Build()         Process()    onFrameCertified()│  │      │    │
│  │  │  │  assign frame    update idx   deliver block     │  │      │    │
│  │  │  │  detect bases    run election                   │  │      │    │
│  │  │  └──────┬──────────────┬─────────────────────────┬─┘  │      │    │
│  │  │         │              │                         │    │      │    │
│  │  │  cheater detection   leader election       epoch mgmt │      │    │
│  │  └──┬──────┼──────────────┼─────────────────────────┼─┬──┘      │    │
│  │     │      │              │                         │ │         │    │
│  │  dagindexer│ vector clock │                         │ │         │    │
│  │  updates   │              │                         │ │         │    │
│  └─────┬──────┼──────────────┼─────────────────────────┼─┼─────────┘    │
│        │      │              │                         │ │              │
│        ▼      │              │                         │ ▼              │
│  ┌────────────┴──────┐       │              ┌──────────┴──────────┐     │
│  │   dagindexer      │       │              │  consensusstore     │     │
│  │                   │       │              │                     │     │
│  │  Index (facade)   │       │              │  Store              │     │
│  │  ┌─────────────┐  │       │              │  ┌───────────────┐  │     │
│  │  │StronglyReach│  │       │              │  │ Bases         │  │     │
│  │  │MedianTime   │  │       │              │  │ EpochState    │  │     │
│  │  │NoCheaters   │  │       │              │  │ Confirmed     │  │     │
│  │  │DfsSubgraph  │  │       │              │  │ LastCertified │  │     │
│  │  └──────┬──────┘  │       │              │  └───────┬───────┘  │     │
│  │         │         │       │              │          │          │     │
│  │    ┌────┴────┐    │       │              └──────────┼──────────┘     │
│  │    │         │    │       │                         │                │
│  │    ▼         ▼    │       │                         │                │
│  │ ┌────────┐┌─────┐ │       │                         │                │
│  │ │dagstore││dag- │ │       │                         │                │
│  │ │        ││branc│ │       │                         │                │
│  │ │ Store  ││h    │ │       │                         │                │
│  │ │ tables ││Track│ │       │                         │                │
│  │ │ caches ││er   │ │       │                         │                │
│  │ └───┬────┘└──┬──┘ │       │                         │                │
│  │     │        │    │       │                         │                │
│  │     └───┬────┘    │       │                         │                │
│  │         ▼         │       │                         │                │
│  │    ┌─────────┐    │       │                         │                │
│  │    │ dagvec  │    │       │                         │                │
│  │    │         │    │       │                         │                │
│  │    │Vector   │    │       │                         │                │
│  │    │ clock   │    │       │                         │                │
│  │    │ types   │    │       │                         │                │
│  │    └─────────┘    │       │                         │                │
│  └───────────────────┘       │                         │                │
│            │                 │                         │                │
│            ▼                 │                         │                │
│     ┌──────────────┐         │                         │                │
│     │vecflushable  │         │                         │                │
│     │              │         │                         │                │
│     │ write-buffer │         │                         │                │
│     │ backed cache │         │                         │                │
│     └──────┬───────┘         │                         │                │
│            │                 │                         │                │
│            ▼                 ▼                         ▼                │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                         kvdb                                    │    │
│  │              Key-Value Store Abstraction (LevelDB)              │    │
│  │                                                                 │    │
│  │  ┌───────────────────┐          ┌───────────────────────┐       │    │
│  │  │    Main DB        │          │   Epoch DB (per epoch) │       │    │
│  │  │  LastCertified    │          │  Bases, Confirmed,     │       │    │
│  │  │  EpochState       │          │  Vector Index          │       │    │
│  │  └───────────────────┘          └───────────────────────┘       │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                                                         │
│  ┌─────────────────────┐                                                │
│  │  consensus (root)   │  Core types: Event, Validators, Block,        │
│  │                     │  EventHash, WeightCounter, EventSource         │
│  │  imported by all    │  (no dependencies on other consensus pkgs)     │
│  └─────────────────────┘                                                │
│                                                                         │
│  ┌─────────────────────┐                                                │
│  │  consensustest      │  Test utilities: TestEvent, ASCII DAGs,       │
│  │                     │  GenNodes, naming helpers                      │
│  │  test-only          │  (depends only on consensus root)              │
│  └─────────────────────┘                                                │
└─────────────────────────────────────────────────────────────────────────┘
```

### Dependency Graph

```
consensus (root types)
    ^       ^       ^       ^       ^
    │       │       │       │       │
    │       │       │       │    consensustest (test-only)
    │       │       │       │
    │     dagvec    │       │
    │      ^ ^      │       │
    │      │ │      │       │
    │ dagstore dagbranch    │
    │      ^       ^        │
    │      │       │        │
    │   dagindexer (root)   │
    │        ^              │
    │        │              │
    │   vecflushable   consensusstore
    │        ^              ^
    │        │              │
    └────────┴──────────────┘
                  │
           consensusengine
```

### Data Flow

```
          Event arrives
               │
               ▼
┌──────────────────────────┐
│  IndexedLachesis.Process │
└──────┬────────┬──────────┘
       │        │
       ▼        ▼
  ┌─────────┐ ┌──────────────────────────────┐
  │dagindex │ │Orderer.Build                  │
  │er.Add() │ │  assign frame (strongly-reach)│
  │ vector  │ │  detect bases                 │
  │ clocks  │ │  run election on bases        │
  └─────────┘ └──────────────┬────────────────┘
                             │ frame certified
                             ▼
              ┌──────────────────────────┐
              │ Lachesis.onFrameCertified │
              │  DFS from leader event   │
              │  detect cheaters         │
              │  BeginBlock → ApplyEvent │
              │  → EndBlock              │
              └──────────────────────────┘
```

## Package Structure

```
consensus/
  *.go                  Core types and interfaces (Event, Hash, Validators,
                          EventSource, BaseDescriptor, etc.)
  consensusengine/      Consensus algorithm: frame assignment, election, finality
  consensusstore/       Persistent storage layer (KV-backed)
  dagindexer/           DAG index facade and algorithms (strongly-reach,
                          median time, cheater filtering)
    dagvec/             Vector clock types and branch data (pure data, no I/O)
    dagbranch/          Branch tracking and equivocation detection
    dagstore/           Vector clock persistence with caching
  vecflushable/         Write-buffered KV wrapper for vector clock storage
  consensustest/        Test utilities, mock events, ASCII DAG visualizations,
                          human-readable naming helpers
```

## Core Types

### Event

The fundamental unit of the DAG. Each event references parent events (one self-parent from the same validator, plus cross-parents from other validators).

```go
type Event interface {
    Epoch() Epoch           // Current epoch
    Seq() Seq               // Sequence number from this creator
    Frame() Frame           // Consensus frame (round)
    Creator() ValidatorID   // Validator that created the event
    Lamport() Lamport       // Lamport timestamp
    Parents() EventHashes   // Parent event references
    ID() EventHash          // 32-byte identifier
    Size() int
}
```

`EventHash` is 32 bytes structured as `[Epoch(4) | Lamport(4) | Hash(24)]`.

### Validators

An immutable, weighted set of validators for an epoch. Provides quorum arithmetic: quorum requires strictly more than 2/3 of total weight.

`WeightCounter` accumulates validator weights and reports `QuorumReached()` (>2/3 weight) and `AntiQuorumReached()` (quorum is impossible given remaining uncounted validators).

### Block

The output of consensus. Contains the leader event hash and a list of detected cheaters (equivocating validators).

```go
type Block struct {
    Leader   EventHash
    Cheaters Cheaters  // []ValidatorID
}
```

## Consensus Protocol

### Key Concepts

**Frames.** Events are assigned to frames (logical rounds). An event advances to a new frame when its parent events from the previous frame are strongly-reachable by a quorum of validators.

**Bases.** An event whose frame is higher than its self-parent's frame is called a *base*. Bases anchor consensus decisions and participate in leader election.

**Strongly-Reach.** Event A *strongly-reaches* event B if more than 2/3 of validator weight has events reachable from A that also reach B. This is the core BFT property ensuring safety. Detected efficiently via vector clock comparison (`HighestBefore >= LowestAfter`).

**Leader Election.** For each frame, an election determines which validator's base event becomes the leader. Votes are aggregated across frame bases using a weighted quorum formula. When a frame is certified (enough votes exceed the threshold), the leader is confirmed.

### Processing Flow

```
Event arrives
    |
    v
Build(event)
    Assign frame number based on parents and strongly-reach quorum.
    If frame > self-parent's frame, mark event as a base.
    |
    v
Process(event)
    Update DAG index (vector clocks).
    If base: store it and run election via VoteAndAggregate().
    |
    v
Election certifies frame
    onFrameCertified(frame, leader)
    |
    v
Deliver block
    BeginBlock(Block{leader, cheaters})
    DFS from leader -> ApplyEvent() for each confirmed event
    EndBlock() -> may trigger epoch seal
    |
    v
Persist
    Update LastCertifiedFrame.
    If epoch sealed: drop old epoch DB, open new one.
```

### Epoch Transitions

Epochs partition consensus into segments, each with its own validator set. When `EndBlock` returns a new validator set, the current epoch is sealed: the epoch DB is dropped, a new one is opened, and the election resets.

## Component Details

### consensusengine/

**Orderer** — Central engine. Processes events, calculates frames, manages bases, and drives the election. Key methods: `constructEventFrame()`, `stronglyReachableByQuorum()`, `runElectionOnBase()`, `onFrameCertified()`.

**Lachesis** — Wraps Orderer. Adds event confirmation tracking, cheater detection (via DFS traversal from the leader), and application callbacks (`BeginBlock`, `ApplyEvent`, `EndBlock`).

**IndexedLachesis** — Wraps Lachesis with a `dagindexer.Index`. Manages vector clock updates on each `Process()` call and handles `DropNotFlushed()` on failure.

**election** — Implements the leader election algorithm. Maintains a vote matrix per frame. `VoteAndAggregate()` processes base votes and checks certification. `elect()` picks the final leader with a tiebreaker for equivocal validators.

**leaderHeap** — Min-heap of certified leaders by frame. `getDeliveryReadyLeaders()` returns a continuous sequence from the next frame to deliver.

### dagindexer/

Split into four layers. `dagvec` is the foundation; `dagstore` and `dagbranch` are sibling layers that both depend on `dagvec`; the root composes them.

```
dagvec  (pure types, no I/O)
  ^  ^
  |  |
dagstore  dagbranch   (siblings, no mutual dependency)
  ^       ^
  |       |
dagindexer root       (facade + algorithms)
```

**dagvec/** — Pure data types with no storage dependency. Contains vector clock types (`HighestBeforeSeq`, `LowestAfterSeq`, `HighestBeforeTime`, `LowestAfter`, `HighestBefore`, `AllVecs`) and vector operations (`InitWithEvent`, `Visit`, `CollectFrom`, `GatherFrom`).

**dagstore/** — Persistence layer. `Store` struct owns KV tables and caches for vector clocks. Provides `Get`/`Set` methods for vector types, event-to-branch mappings, and raw byte storage for branch metadata. Implements `dagbranch.BranchStore` (verified by compile-time check in `compat.go`).

**dagbranch/** — Branch tracking and equivocation detection. Owns the `BranchesInfo` type and its RLP serialization. `Tracker` manages the in-memory `BranchesInfo` state, assigns branch IDs to new events (`AssignBranchID`), detects equivocations by cross-branch sequence comparison (`DetectEquivocations`), and merges scattered per-branch vectors into per-validator vectors (`GetMergedHighestBefore`). Accesses persistence through a `BranchStore` interface.

**Root package** — `Index` facade composes `*dagstore.Store` and `*dagbranch.Tracker`, and exposes algorithms:
- `StronglyReach()` — BFT strongly-reach detection via vector clock comparison (`HighestBefore >= LowestAfter`), accounting for equivocal branches.
- `MedianTime()` — Weighted median of `HighestBeforeTime` across validators.
- `NoCheaters()` — Filters detected equivocators from a validator set.
- `DfsSubgraph()` — DAG traversal utility.

Type aliases in `compat.go` re-export `dagvec` types so that consumers (e.g. `consensusengine`) require no import changes.

### vecflushable/

A write-buffered wrapper around `kvdb.Store` optimized for vector clock access patterns.

- `modified` map: in-memory write buffer (not yet flushed)
- `backedMap`: bounded in-memory cache backed by persistent storage
- `Flush()` persists buffered writes; `DropNotFlushed()` discards them
- Automatic eviction when cache exceeds `maxMemSize`

### consensustest/

Test utilities including `TestEvent` (minimal Event implementation), `TestEventSource` (in-memory event storage), `GenNodes()` (validator generation), ASCII DAG visualization for readable test cases, and human-readable naming helpers (`SetNodeName`/`GetNodeName`, `SetEventName`/`GetEventName`) for mapping hashes to short names in test output.

## Application Integration

The library requires two interfaces from the host application, both defined in the root `consensus` package:

```go
// Provides event data to the consensus engine (consensus/event_source.go)
type EventSource interface {
    HasEvent(EventHash) bool
    GetEvent(EventHash) Event
}

// Receives consensus decisions (consensus/consensus.go)
type ConsensusCallbacks struct {
    BeginBlock func(block *Block) BlockCallbacks
}

type BlockCallbacks struct {
    ApplyEvent func(event Event)      // Called per confirmed event
    EndBlock   func() *Validators     // Non-nil triggers epoch seal
}
```

## Database Format

The storage layer uses a key-value store abstraction (`kvdb.Store`) with LevelDB as the backend. Data is split between a persistent main DB and per-epoch databases.

### Main DB

Global state that persists across epochs:

| Table Prefix | Key | Value | Description |
|---|---|---|---|
| `d` | `"d"` | RLP(`LastCertifiedState`) | Last certified frame number |
| `e` | `"e"` | RLP(`EpochState`) | Current epoch number and validator set |

### Epoch DB

Created fresh for each epoch and dropped on epoch transitions:

| Table Prefix | Key | Value | Description |
|---|---|---|---|
| `r` | `Frame(4B) \| ValidatorID(4B) \| EventHash(32B)` | `[]byte{}` (empty) | Frame bases index |
| `C` | `EventHash(32B)` | `Frame(4B)` | Event confirmation frame |
| `v` | *(sub-tables below)* | | Vector clock index |

The vector index table (`v`) is wrapped by `VecFlushable` and managed by `dagstore.Store`. It contains sub-tables:

| Sub-Prefix | Key | Value | Description |
|---|---|---|---|
| `T` | `EventHash(32B)` | `[8B per validator]` LE uint64 timestamps | HighestBeforeTime vectors |
| `S` | `EventHash(32B)` | `[8B per validator]` LE uint32 Seq + uint32 MinSeq | HighestBeforeSeq vectors |
| `s` | `EventHash(32B)` | `[4B per validator]` LE uint32 Seq | LowestAfterSeq vectors |
| `b` | `EventHash(32B)` | `ValidatorIndex(4B)` BE | Event-to-branch mapping |
| `B` | `"c"` | RLP(`BranchesInfo`) | Branch metadata for equivocation tracking |

### Serialization

- **RLP encoding** for structured data: `LastCertifiedState`, `EpochState`, `BranchesInfo`
- **Raw binary** for vector clocks (fixed-size elements at computed offsets)
- **Big-endian uint32** for all index types: `Epoch`, `Frame`, `Seq`, `ValidatorID`, `ValidatorIndex`
- **Little-endian** for vector clock elements within binary-encoded vectors

### Caching

Multiple cache layers reduce DB reads:

| Cache | Type | Default Size | Scope | Owner |
|---|---|---|---|---|
| Frame bases | simplewlru | by entry count | Per epoch, purged on transition | consensusstore |
| HighestBeforeSeq | simplewlru | 160 KB | Per epoch | dagstore.Store |
| HighestBeforeTime | wlru | 160 KB | Per epoch | dagstore.Store |
| LowestAfterSeq | simplewlru | 160 KB | Per epoch | dagstore.Store |
| VecFlushable | backedMap | 10 MiB | Per epoch | vecflushable |
| LastCertifiedState | direct | 1 entry | Global | consensusstore |
| EpochState | direct | 1 entry | Global | consensusstore |

### Storage Lifecycle

```
NewStore(mainDB, epochDBProducer)
    |
    v
ApplyGenesis(epoch, validators)     Write initial EpochState and LastCertifiedState
    |
    v
OpenEpochDB(epoch)                  Create epoch DB, migrate tables, init caches
    |
    v
... Process events ...              Writes to bases, vector index, confirmed events
    |
    v
Epoch sealed:
    DropEpochDB()                   Close and delete old epoch DB
    OpenEpochDB(newEpoch)           Fresh epoch DB for new validator set
```
