# Amber

Amber is a programming language built around one idea: **references should come from
the data they point to, not from the machine running the program.**

In every other language, a reference is essentially a memory address — a number
that says "this value lives at slot 4,827 on this machine, right now." That number
is meaningless tomorrow. It is meaningless on your colleague's machine. It is
meaningless after a restart. The entire category of bugs called stale pointers,
cache invalidation failures, desynchronization, and "works on my machine" exists
because references and data are two separate things that can fall out of step.

Amber makes references a fingerprint of the data itself — a short string computed
from what a value actually contains. The same value always produces the same
fingerprint. A different value always produces a different fingerprint. That
fingerprint works everywhere: in memory, on disk, across a network, in a file you
share with a colleague, in a registry on the other side of the world. You never
have to translate it, version it, or rebuild it. It just works.

Everything else in Amber follows from taking that seriously.

---

## Contents

- [How References Work](#how-references-work)
- [Values Don't Change](#values-dont-change)
- [Why Restarts Become Delays](#why-restarts-become-delays)
- [Sharing Is Sending a Reference](#sharing-is-sending-a-reference)
- [The Object Store](#the-object-store)
- [Data Model — Two Tiers](#data-model--two-tiers)
- [Numbers and Canonical Encoding](#numbers-and-canonical-encoding)
- [Syntax](#syntax)
- [Processes](#processes)
- [Freezing and Thawing](#freezing-and-thawing)
- [Supervision](#supervision)
- [Connecting Runtimes](#connecting-runtimes)
- [Names and Packages](#names-and-packages)
- [Types](#types)
- [Consequences of This Model](#consequences-of-this-model)
- [Implementation Plan](#implementation-plan)
- [Getting Started](#getting-started)
- [Project Structure](#project-structure)
- [Contributing](#contributing)

---

## How References Work

Amber uses [BLAKE3](https://github.com/BLAKE3-team/BLAKE3) to compute a fingerprint
for every value. BLAKE3 is fast — around 3 GB/s on a single core — and produces a
32-byte output that is unique for all practical purposes.

A fingerprint in Amber is written with a `#` prefix:

```
#a3f9c2e1b8d4f7a2c9e6b3d0f8a5c2e9b6d3f0a7c4e1b8d5f2a9c6e3b0d7f4a1
```

You rarely write these by hand. You write names. But underneath every name is a
fingerprint, and that fingerprint is what Amber actually uses.

The fingerprint is computed from the **canonical encoding** of the value — a
deterministic byte representation where:

- Object keys are sorted alphabetically
- Floats follow JSON normalization rules (`4.000 → 4.0`)
- Strings are UTF-8 NFC normalized
- Integers are fixed-width little-endian
- Absent optional fields are represented explicitly, never omitted

Same value → same canonical bytes → same BLAKE3 fingerprint. This is not a
convention. The runtime enforces it and relies on it.

---

## Values Don't Change

In Amber, values are immutable. There is no operation that modifies an existing value.
When you want a "changed" version of something, Amber produces a new value — a new
object with a new fingerprint — leaving the original untouched.

```js
const point = { x: 3, y: 4 }
// fingerprint: #a3f9c2...

const moved = { ...point, x: 5 }
// fingerprint: #bb21f4...  — a different value, a different fingerprint
// point is unchanged. Both exist. Either can be referenced.
```

This is not wasteful. When two values share structure, they share the underlying
data. Only the parts that actually changed exist as new data. The rest is referenced
from the original. A large object where one field changed takes roughly the space of
that one change plus a small amount of structural overhead, not the space of a full
copy.

The practical consequence is that the entire history of your values is naturally
preserved. Last Tuesday's dataset and today's dataset can exist simultaneously.
You can compare them, branch from either, share either. You don't build version
control on top — you already have it by default.

---

## Why Restarts Become Delays

Most software rebuilds its working model of the world every time it starts. It reads
saved files, reconstructs data structures, builds a fresh set of references for this
particular run on this particular machine. This is what a restart actually does: it
flushes all the references that were valid for the last run and builds new ones for
this run.

This is necessary because references are tied to the running program. When the program
stops, the references stop being valid. A crash loses any work that hadn't been saved.
A migration that fails partway through leaves some data in the old form and some in
the new form, with no clean way to know which is which.

In Amber, references are not tied to a running program. They are derived from the data.
When a runtime stops — crash, power failure, deliberate shutdown — the references in
the object store remain valid. When a runtime starts again, it does not rebuild
anything. The data is there, correctly referenced, and work resumes from wherever
it was.

A restart is a delay. Not a reset. Not a loss of work. Just a pause while the runtime
process comes back up.

Migrations in Amber produce new references for new forms. The old references continue
to work for the old form. There is no in-between state, no inconsistency window, no
recovery procedure for a migration that got interrupted. Old and new coexist until you
choose to discard the old.

---

## Sharing Is Sending a Reference

In most systems, sharing data between machines is a two-step problem. Move the bytes
across the network. Rebuild the internal references on the other side. Now you have
two copies. They start synchronized. They drift the moment either side changes anything.
Keeping them in sync is a standing engineering problem.

In Amber, a reference is the same everywhere. When you send someone a reference, they
either already have the value (because they computed or received it before), or they
ask you to send the bytes once. Either way, they end up with the same reference you
have, pointing to the same value. There are no two copies. Drift is not a problem you
manage — it is a condition that cannot arise, because two machines holding the same
reference are holding the same data by definition.

This changes what sharing looks like:

**Sharing results.** You run an analysis. The result has a fingerprint. You send the
fingerprint to a colleague. If they already have it, they have the result instantly.
If not, they fetch it once. They don't need to re-run your computation. They don't
need to understand your data format. They use the fingerprint.

**Sharing bugs.** A crashed program leaves behind a frozen snapshot. You send the
fingerprint. Your colleague loads it. They are in the exact state your program was
in when it crashed — not an approximation, not a reproduction attempt, the actual
state.

**Sharing code.** A script has a fingerprint. You share the fingerprint. Your
colleague runs exactly the script you ran, not a version that might have drifted,
not a copy that might have been modified. The fingerprint guarantees it.

References in Amber behave like URLs that actually work the way URLs were supposed to
work — stable, location-independent, guaranteed to refer to exactly what they say
they refer to, forever.

---

## The Object Store

Every Amber runtime maintains a local object store: a key-value database where keys
are BLAKE3 fingerprints and values are the canonical byte representations of Amber
values.

When a value is computed or received, it goes into the store. When a value is
referenced, it is retrieved from the store. Values are never deleted automatically —
the store grows until you explicitly prune it. Pruning is safe: any value whose
fingerprint is not referenced by anything you care about can be removed.

Runtimes share objects over the network by fingerprint. Before sending a value to a
peer runtime, Amber checks which fingerprints the peer already has. Only novel objects
are transmitted. Over time, runtimes that exchange data frequently converge toward a
shared working set, and transfers approach zero marginal cost for stable data.

A **registry** is a runtime configured to be publicly reachable with high storage
limits. It is not special software — just an Amber runtime set up to accept objects
from many sources and serve them by fingerprint. You can run your own. The objects
you store on a registry are retrievable by anyone with the registry's address and
the fingerprint.

---

## Data Model — Two Tiers

Amber stores all data in one of two forms. The form is determined by the shape of
the data.

### Tier 1 — Flat Tables (preferred)

A table is a schema and a blob.

The **schema** names the columns, their types, and their order. The schema is itself
a value with a fingerprint. It is defined once and reused by every table with the
same layout.

The **blob** is raw packed binary data — the column values for every row, laid out
contiguously in memory. No pointers. No indirection.

The fingerprint of a table is computed in one pass over the blob, with the schema
fingerprint prepended. No pointer chasing. No tree traversal. Fast.

```js
const Position = schema({ x: "f32", y: "f32", z: "f32" })
// fingerprint: #a3f9c2...  — shared by all position tables

const positions = table(Position, [
  { x: 1.0, y: 2.0, z: 3.0 },
  { x: 4.0, y: 5.0, z: 6.0 },
  // ...100,000 rows
])
// fingerprint: one BLAKE3 pass over schema_fingerprint + raw bytes
```

This tier is for structured data where the shape is known and uniform: records,
time series, entity attributes, configuration rows. If the data fits a fixed schema,
it belongs here.

Column types:

| Type | Description |
|---|---|
| `u8` `u16` `u32` `u64` | Unsigned integers |
| `i8` `i16` `i32` `i64` | Signed integers |
| `f32` | 32-bit float, JSON-normalized |
| `bool` | Single byte |
| `str` | Variable-length UTF-8, packed with offset table |
| `bytes` | Variable-length binary, packed with offset table |
| `ref` | Explicit fingerprint pointer into Tier 2 |

The presence of `ref` in a schema is intentional and visible. If you see `ref`, you
are explicitly opting into the pointer graph. Everything else stays flat.

### Tier 2 — Document Graph

Used for recursive and variable-structure data: nested documents, code objects,
process frames, abstract syntax trees. A document is a map from field names to
fingerprints of other values. Its fingerprint is computed from its canonical encoding.
Fingerprints are cached after first computation.

```js
const config = {
  server: {
    host: "localhost",
    port: 7700
  },
  limits: {
    maxConnections: 100
  }
}
// Each nested object has its own fingerprint.
// The outer fingerprint is computed from the inner fingerprints.
```

### Choosing a Tier

The right signal: if you are reaching for `ref` in a table schema where the
referenced type is always the same, you probably want a second flat table with an
integer key column instead. Flat tables are faster to hash, cheaper to transmit,
and better for performance-sensitive code.

Data that is genuinely recursive or variable in structure belongs in Tier 2.
The two tiers exist because one representation does not fit everything.

---

## Numbers and Canonical Encoding

### Integers

Full range unsigned and signed integers from 8 to 64 bits. Sizes are explicit.
No silent promotion. No coercion between integer types. Explicit casts only.

### Floats

One floating-point type: `f32`. All floats follow JSON normalization:

- `4.000` becomes `4.0`
- `1.50` becomes `1.5`
- Trailing zeros are dropped
- Negative zero is normalized to zero
- `NaN` and `Infinity` are not valid Amber values — computations producing them are errors

For cases requiring more precision or exactness:

**`Decimal`** — arbitrary-precision decimal. Backed by string representation. Exact.
Slower than `f32`. For money, user-facing measurements, anything where rounding
must not occur silently.

**`Fraction`** — exact rational as `{ numerator: i64, denominator: i64 }`. For
symbolic computation or proportional layout where decimal approximation is wrong.

No implicit promotion between `f32`, `Decimal`, and `Fraction`. You choose the type
that matches the domain.

---

## Syntax

Amber uses a subset of JavaScript syntax. The semantics underneath are different,
but the notation is familiar to anyone who has written modern JS.

### What Amber Keeps

```js
// Constant bindings — the only kind
const greeting = "hello"
const point = { x: 3, y: 4 }

// Arrow functions
const add = (a, b) => a + b
const double = x => x * 2

// Object and array literals
const config = { host: "localhost", port: 7700 }
const primes = [2, 3, 5, 7, 11]

// Spread — the primary way to derive new values
const moved = { ...point, x: 5 }
const extended = [...primes, 13]

// Destructuring
const { x, y } = point
const [first, ...rest] = primes

// Template literals
const message = `connecting to ${config.host}:${config.port}`

// Async/await (syntax only — backed by frozen continuations)
const result = await fetchFrom(peer, fingerprint)

// Import and export (resolved to fingerprints at build time)
import { map, filter } from "stdlib/list"
export const parseDate = input => ...
```

### What Amber Removes

- `let` and `var` — no mutable bindings. `const` only.
- `class`, `new`, `this`, `prototype` — no classes, no inheritance.
- `null` — one absence value: `undefined`.
- `typeof`, `instanceof` — replaced by schema predicates.
- Numeric coercion — no silent `"3" + 4`. Explicit casts.
- Getter and setter traps — plain values only.
- `for`, `while`, `do...while` — see below.
- `arguments` — arrow functions only.

### Iteration — Bounded Only

Amber supports only bounded iteration over finite collections. No `while` loop,
no general `for` loop. This guarantees termination for data-processing code.

```js
// The only loop form
items.forEach((item, index) => {
  // ...
})

// Standard library iteration
const doubled   = items.map(x => x * 2)
const evens     = items.filter(x => x % 2 === 0)
const total     = items.reduce((sum, x) => sum + x, 0)
const firstBig  = items.find(x => x > 100)
```

Unbounded computation belongs in processes (see below), which are explicitly
long-running actors. The distinction is visible and intentional: data
transformations terminate, processes run until stopped.

### Pattern Matching

```js
const response = match(message) {
  { type: "ok",    value }      => handleValue(value),
  { type: "error", code: 404 }  => handleNotFound(),
  { type: "error", code }       => handleError(code),
  _                             => ignore()
}
```

Guards on patterns must be pure functions:

```js
match(entity) {
  { health } if health <= 0  => die(),
  { health } if health < 20  => flee(),
  _                          => fight()
}
```

### Added Syntax

```js
// Fingerprint literals — reference a known value directly
const knownFn = #a3f9c2e1b8d4f7a2...

// Schema definitions
const Vec3 = schema({ x: "f32", y: "f32", z: "f32" })

// Schema predicates
if (Vec3.matches(v)) { ... }

// Spawn, freeze, thaw, receive — process builtins
const pid    = spawn(loopFn, initialArgs)
const frozen = freeze(currentThread)
const resumed = thaw(frozenFingerprint)

receive {
  { type: "ping" } => send(sender, { type: "pong" }),
  { type: "stop" } => exit()
}
```

---

## Processes

A process is a long-running computation with a mailbox. Processes communicate by
sending messages. A message is a fingerprint — the receiver retrieves the value
from the object store.

### Process Identity

A process is identified by the fingerprint of its defining function plus its initial
arguments:

```
pid = BLAKE3(function_fingerprint ++ initial_args_fingerprint)
```

This means:
- The same function started with the same arguments always has the same identity,
  on any runtime, at any time.
- Spawning a process that already exists (same identity) is idempotent.
- You can compute a process identity from first principles without querying a registry.
- A process identity is stable and can be stored, compared, and shared.

```js
// A simple counter process
const counterLoop = (state) =>
  receive {
    { type: "inc" }       => counterLoop({ ...state, count: state.count + 1 }),
    { type: "get", from } => {
      send(from, state.count)
      counterLoop(state)
    },
    { type: "stop" }      => state.count
  }

const counter = spawn(counterLoop, { count: 0 })
send(counter, { type: "inc" })
send(counter, { type: "get", from: self })
```

### Value Cache

Because process identity is a fingerprint of function plus initial arguments, a
runtime can recognize when it has already run an identical process to completion.
If the result of a process is already in the object store from a previous run, the
runtime may return the cached result without spawning again.

This is declared in the process spec and opt-in. It turns deterministic long-running
computations into memoized function calls transparently.

### Selective Receive

A process can wait for a specific kind of message while leaving others in the mailbox:

```js
// Only handle urgent messages now; leave others queued
receive {
  { priority: "urgent", payload } => handleUrgent(payload)
}
// Non-urgent messages remain in the mailbox for the next receive
```

---

## Freezing and Thawing

A running process — its code, its current position, its local data, its pending
messages — can be captured as a value with a fingerprint. This is called freezing.

```js
const snapshot = freeze(currentProcess)
// snapshot is a fingerprint
// everything needed to resume is in the object store under that fingerprint
```

A frozen process can be:
- Saved to the object store (automatically — it is a value)
- Sent to another runtime by fingerprint
- Resumed on any runtime that has the object store entry

```js
const resumed = thaw(snapshot)
// execution continues exactly where it left off
// on any runtime, after any amount of time
```

This is not a special migration feature or backup system. It falls out from the
same principle that governs all values: a process is a value, values have
fingerprints, fingerprints work everywhere.

**What a frozen process contains:**

```js
{
  stack:  [ frame, frame, ... ],   // current call frames
  heap:   fingerprintOfData,       // all values the process references
  pc:     fingerprintOfNextStep,   // where execution resumes
  inbox:  [ msg, msg, ... ]        // messages waiting to be processed
}
```

The entire structure fingerprints to a single root fingerprint. Sharing a frozen
process means sharing that fingerprint. The receiving runtime fetches whatever
objects it doesn't already have.

**Continuations as return addresses.** When a runtime sends work to a peer and
needs the result back, it does not block with an open connection. It freezes its
current continuation (the code to run when the result arrives) and attaches the
fingerprint to the outgoing job. The peer runs the work, deposits the result, and
sends it to the continuation fingerprint. The original runtime thaws the continuation
with the result in hand. No thread blocks. No socket held open. Waiting is frozen
state in the object store.

---

## Supervision

Processes are supervised. A supervisor is a process that watches other processes and
restarts them when they crash.

```js
const supervisor = makeSupervisor({
  strategy: "one_for_one",   // restart only the crashed child
  children: [
    { fn: workerFn, args: workerArgs, checkpointEvery: 100 }
  ]
})
```

Strategies:
- `one_for_one` — restart only the crashed process
- `one_for_all` — restart all children when one crashes
- `rest_for_one` — restart the crashed process and all processes started after it

When a child crashes, the supervisor receives a crash record:

```js
{
  pid:             fingerprintOfProcess,
  reason:          errorValue,
  lastCheckpoint:  fingerprintOfLastFrozenState
}
```

The supervisor can inspect the crash, decide whether to restart, and restart from
the last checkpoint rather than from scratch.

---

## Connecting Runtimes

Runtimes connect to each other by address. Once connected, they exchange fingerprints
and transfer objects on demand.

```js
const peer = connect("amber://hostname:7700")

// Send a value — transmits only objects the peer doesn't have
send(peer, myData)

// Request a value by fingerprint
const result = await fetch(peer, #a3f9c2...)
```

Connections are symmetric. Either side can request objects from the other. The
protocol is content-addressed: both sides refer to all values by fingerprint, and
the wire format is just fingerprint + canonical bytes for novel objects.

A runtime can be configured to **pin** certain fingerprints — keep them available
even under pruning pressure. A pinned fingerprint is always servable to peers.

---

## Names and Packages

Names in Amber are aliases for fingerprints. A name resolves to exactly one
fingerprint at any point in time. The resolution is recorded in the object store
itself, so the mapping is versioned and auditable.

```js
// A name binding
name stdlib/list = #a3f9c2...

// Importing by name — resolved to fingerprint at build time
import { map } from "stdlib/list"
// After resolution:
import { map } from #a3f9c2...
```

A package is a named collection of exports — a document whose fields are names and
whose values are fingerprints. Package versions are themselves values: a version bump
produces a new fingerprint for the package document.

There is no package manager in the traditional sense. A package is a fingerprint you
can request from a registry. Pinning a version means recording a fingerprint. Updating
means recording a new fingerprint. Rollback means reverting to the old one.

---

## Types

Amber's type system is structural and schema-based.

### Schema as Type

A schema definition is the primary way to define a type:

```js
const Vec3    = schema({ x: "f32", y: "f32", z: "f32" })
const Particle = schema({ pos: Vec3, vel: Vec3, mass: "f32" })
```

Schemas compose. A schema field can reference another schema. The composed schema's
fingerprint incorporates the nested schema's fingerprint.

### Type Inference

Within a function body, types are inferred from use. A function that calls `.map`
on its argument is inferred to accept a type that supports `.map`. A function that
spreads an object with `{ ...v, x: 5 }` is inferred to return a type with at least
`x: f32` (or the inferred type of `5`).

### Union Types

```js
const Result = union({
  ok:    { value: "ref" },
  error: { code: "u32", message: "str" }
})
```

Pattern matching is exhaustive over union members. A `match` that does not cover all
members is a compile-time error.

### Gradual Typing

Amber allows unannotated code. Unannotated functions are checked at call sites.
If a call site provides enough type information to resolve all operations in the
function body, the call is accepted. If not, the compiler reports the ambiguity.

Full annotations are optional but produce better error messages.

---

## Consequences of This Model

A few things that follow directly from content-addressed values and immutable data:

**Reproducibility is structural.** Two programs given the same input fingerprints
produce the same output fingerprints. Reproducibility is not a practice you enforce
— it is a property of the model.

**Caching is exact.** A cached result is valid as long as the input fingerprints
haven't changed. They can't change — values are immutable. Cached results never
go stale. Cache invalidation is not a problem you solve.

**Auditing is free.** Every transformation produces a new value with a new fingerprint.
The fingerprint of any output traces back to the fingerprints of its inputs. You
can follow the chain of transformations from any result back to its source data.
Audit trails are not infrastructure you build — they are a side effect of how values work.

**Diffs are structural.** Two values with a common ancestor share all the fingerprints
that haven't changed. A diff is a set of fingerprints that are in one value but not
the other. Structural diff and merge don't require understanding the data format.

**Testing is property-based by default.** A function that takes fingerprints and
returns fingerprints can be tested with any values you have in your object store.
Test fixtures are fingerprints. Regressions are fingerprints. Property tests operate
over the same value space as production.

---

## Implementation Plan

The implementation proceeds in layers, each building on the one before.

### Layer 0 — Canonical Encoding and Fingerprinting

The foundation. Implement:

- Canonical encoder for all Amber value types
- BLAKE3 fingerprint computation
- Object store (in-memory first, then persistent)
- Basic serialization/deserialization

Nothing else works without this. Get it right before moving on.

### Layer 1 — Lexer and Parser

Parse Amber source into an AST. The AST is itself an Amber value (stored in the
object store with a fingerprint). This means the parser's output is directly usable
as data by the rest of the system.

### Layer 2 — Evaluator

A tree-walking interpreter over the AST. Not fast. Fast enough to test the model.
Implement:

- Constant bindings
- Arrow functions and application
- Object and array literals
- Spread and destructuring
- Pattern matching
- Schema definitions and predicates

No processes yet. Pure data transformations only.

### Layer 3 — Processes and the Runtime

Add:

- Process spawning and identity
- Message passing (in-process first)
- Selective receive
- Freeze and thaw (serialize/deserialize a process to an Amber value)
- Basic supervision

This is the point where restarts become delays rather than resets.

### Layer 4 — Networking

Connect two runtimes. Implement:

- Object transfer protocol (fingerprint exchange, novel-object transmission)
- Remote spawn and remote send
- Registry support

### Layer 5 — Compiler

Replace the tree-walking evaluator with a compiler to a bytecode VM or native code.
The object store's fingerprints mean the compiler's output is content-addressed:
the compiled artifact for a given source fingerprint is itself a value with a
fingerprint. Incremental compilation is caching.

---

## Getting Started

> **Note:** Amber is in early development. The implementation is not yet complete.
> The sections below describe the intended development setup.

### Prerequisites

- [Go](https://go.dev/) 1.22 or later

### Install

```sh
git clone https://github.com/shanecandoit/Amber-language.git
cd Amber-language
go mod tidy
```

### Build

```sh
go build -o amber ./cmd/amber
```

### Lex a file (Layer 1 — available now)

```sh
./amber lex examples/hello.amber
```

### Run a file (not yet implemented)

```sh
./amber examples/hello.amber
```

### Test

```sh
go test ./...
```

---

## Project Structure

```
Amber-language/
├── cmd/
│   └── amber/          # CLI entry point (go build → ./amber)
├── internal/
│   ├── encoding/       # Canonical encoder, BLAKE3 fingerprinting, value types
│   ├── store/          # Object store (key: fingerprint, value: bytes)
│   ├── lexer/          # Tokenizer — source text → token stream
│   ├── parser/         # Parser — token stream → AST  (Layer 1, in progress)
│   ├── evaluator/      # Tree-walking interpreter     (Layer 2, planned)
│   └── runtime/        # Process model, freeze/thaw   (Layer 3, planned)
├── examples/           # Example Amber programs
└── docs/               # Additional documentation
```

---

## Contributing

Amber is in its early stages. The best way to contribute right now is to read the
spec, run the examples, and open issues for anything that seems wrong, underspecified,
or missing.

If you want to contribute code, start with Layer 0 (canonical encoding and
fingerprinting). Everything depends on it being correct.

---

## License

See [LICENSE](LICENSE).
