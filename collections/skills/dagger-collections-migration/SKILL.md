---
name: dagger-collections-migration
description: Migrate a Dagger module that returns lists of objects (with per-item checks or generators) to Dagger collections — keyed dimensions, batch functions, deltas. Use when converting `[Item!]!` fields to `@collection` types, adding a nested dimension, replacing aggregate "run everything" checks with batches, or debugging collection discovery, selection and listing. Examples are Dang; the rules apply to every SDK.
---

# Migrating a module to collections

A collection is an object type with unique keys and a lookup. The engine turns
every field that returns one into a **dimension**, so users can list keys,
select items (`--go-module=sdk/go`), and have checks and generators run over
just the selection — once per collection value, through a **batch** function,
instead of once per item.

This guide comes from migrating `github.com/dagger/go` (`go`, `golangci-lint`,
`staticcheck`, and the shared `gomod` library). Read "Gotchas" before you
start; most of the time spent on that migration went there.

## When a field should become a collection

Convert a field when all of these hold:

- It returns a list of objects that have a stable, unique identity (a path, a
  name) — that identity becomes the key.
- Users want to select some items by that identity.
- The items carry `@check` / `@generate` functions, or will.

Keep a plain list when the items are internal plumbing, or when exposing their
checks would duplicate work another check already does (see "Nested
collections" below). A raw list is not traversed by discovery, so it is also
the way to keep item checks *out* of `dagger check`.

## The shape

```dang
"""
Go modules in a workspace, keyed by module root.
"""
type GoModules @collection {
  """
  Module root paths.
  """
  paths: [String!]! @keys          # stored field: non-null list of scalars

  let mountPath: String!           # only what get needs; never a Workspace
  let tool: Go!                    # back-reference to settings, private

  """
  The Go module rooted at path.
  """
  module(path: String!): GoModule! @get {   # exactly one arg, the key type
    tool.goModuleAt(path, mountPath)
  }

  """
  Run tests in the selected Go modules.
  """
  test(ws: Workspace!): Void @check {       # a batch function
    let failures = paths.map { p => module(p) }.{{path, failure: testFailure(ws)}}
      .filter { r => r.failure != null }
    if (failures.length > 0) {
      raise "Go module tests failed:\n" + failures.map { r => "- " + r.path + ": " + (r.failure ?? "") }.join("\n")
    }
    null
  }
}
```

- `@collection` is required even with the default member names `keys` / `get`.
  `@keys` and `@get` let you keep domain names (`paths`, `module`).
- Every other exposed function moves under `batch` in the public API.
- Consumers never see your names: they get `keys`, `list`, `get(key:)`,
  `subset(keys:)`, `batch`. Your own module code keeps using `paths` and
  `module(p)`.

## Migration steps

1. **Upgrade the module config, then require engine `v1.0.0-beta.15`.**
   Collections need it, and they need the current config format:
   - A `dagger.json` with no `dagger-module.toml` beside it is a **legacy**
     module. Legacy engines can't run collections, so there is no legacy
     support to carry forward: convert it with `dagger module migrate -y
     <path>`, which writes `dagger-module.toml`, removes `dagger.json`, and
     registers the SDK scope in the workspace `dagger.toml` if there is one.
     Review the diff; it can touch the module's `.gitignore` too.
   - Then set `engineVersion = "v1.0.0-beta.15"` in `dagger-module.toml`.
     The migration keeps the old version, and an older `engineVersion`
     doesn't know `@collection`, `@keys`, `@get` or `CollectionDelta`.

   Do this first. Every consumer of the module now needs a beta.15 engine;
   call it out in the change. beta.15 is not released yet, so the module only
   loads on a dev engine for now.

2. **Declare the collection type** as above. Store the keys, plus whatever
   `get` needs to build an item. `get` takes only the key, so anything else it
   needs must be stored on the collection.

3. **Do not store a `Workspace`.** A `Workspace` field invalidates caching for
   the whole object. Store derived values instead (a mount path string), and
   give batch functions a `ws: Workspace!` argument. The CLI fills it in;
   callers from other modules pass it explicitly.

4. **Return the collection from the parent field**, keeping its filter
   arguments with defaults. Discovery calls it with the defaults and the
   injected `Workspace`:

   ```dang
   modules(ws: Workspace!, include: [String!]! = [], includeSkipTest: Boolean! = true): GoModules! {
     GoModules(
       paths: lib.modules(ws, include: include).{{path}}.map { m => m.path },
       mountPath: workspaceMountPath(ws),
       tool: self,
     )
   }
   ```

   Dang's `or`/`and` short-circuit, so a filter like
   `includeSkipTest or mod.skipTest == false` costs nothing at the default.

5. **Move aggregate checks into batch functions, named after the item
   function they replace.** `testAll` on the parent becomes `test` on the
   collection because the item has `test`. Then **delete the aggregate** — once
   the list is a collection, discovery finds the item checks too, and both run.
   A batch with no matching item function still works; it runs once on the
   selection under its own name.

6. **Write the batch body over the keys.** The engine hands the batch a copy
   of the collection holding only the selected keys. Build items with your own
   `get` (`paths.map { p => module(p) }`) — the standard `list` is not visible
   from inside. A selection over a list (`.{{…}}`) evaluates concurrently, which
   is how to run the per-item work in parallel.

7. **Update callers.** From another module or a test suite:

   | Before | After |
   |---|---|
   | `tool.modules(ws).{{path}}` | `tool.modules(ws).keys` |
   | `tool.testAll(ws)` | `run(tool.modules(ws).batch.test(ws))` |
   | pick one by path | `tool.modules(ws).get(key: p)` |
   | filter a list | `tool.modules(ws).subset(keys: [a, b])` |

8. **Wrap check calls from dependencies.** A `@check` called through a
   dependency returns a `Check!` that has not run — as a statement it does
   nothing. `Check.error` is also non-null for a passing check when read from
   Dang. Use `pass`:

   ```dang
   let run(check: Check!): Void {
     if (check.pass == false) {
       raise check.error.message ?? "check failed"
     }
     null
   }
   ```

   Plain `Void` functions (no `@check`) still run eagerly as statements.

9. **Rename in docs and CI.** Check addresses change: `go:test-all` becomes
   `go/modules/test`, selected as `--go --test`. Anything selecting checks by
   name or address needs updating.

10. **Rewrite docstrings for where they now appear.** Public docstrings are
    shown to CLI users; keep them to a line and put the detail in the README.
    Collections change which ones users see:

    - The **item** check's docstring labels the line in `dagger check -l`,
      even when the batch is what runs. Write it for one item: "Run this
      test.", "Run golangci-lint in this module."
    - The batch's docstring shows only in the API; the same sentence over the
      selection is enough: "Run golangci-lint in the selected Go modules."
    - The `@get` docstring is each item's description in `dagger list`.
    - A shared verb (`--lint`) lists several modules' checks side by side, so
      word the same concept the same way across sibling modules.
    - Don't restate what users assume ("respecting the configured selection").
      Selection is the engine's job now.

## Nested collections

A collection field on the item type adds a second dimension under the first
(`GoModule.tests` → `go-test` under `go-module`). Addresses carry one key per
dimension along the path, and a batch runs once **per parent item**, never
across parents.

Prefer nesting when the child belongs to the parent (tests belong to a
module; `go test` runs inside one module). Siblings on one type
(`GoModule.tests` beside `GoModule.benchmarks`) are independent paths: no
address carries both keys, and combining their flags matches nothing.

### The double-run trap

A parent-level check and a nested batch over the same work both run under a
bare `dagger check`. Batch replacement only works sideways (same name, same
collection); nothing tells the engine that `GoModule.test` covers
`GoModule.tests.test`.

Fix it with one hierarchy plus a delta:

```dang
type GoTests @collection {
  names: [String!]! @keys
  delta: CollectionDelta               # the engine fills this in
  let module: GoModule!

  function(name: String!): GoTest! @get { GoTest(name: name, module: module) }

  test(ws: Workspace!): Void @check {
    # Nothing filtered out: run the whole module natively. Without a delta,
    # the explicit list is the safe choice.
    let whole = if (delta == null) { false } else {
      delta.addedKeys.length == 0 and delta.removedKeys.length == 0
    }
    if (whole) { module.test(ws) } else if (names.length > 0) { module.runTests(ws, names) }
    null
  }
}
```

Then drop `@check` from the parent-level function (keep it as a plain
function for API callers). Only declare a delta where the batch has a cheaper
or broader whole-collection operation; per-module batches that must run each
item in its own container gain nothing from one.

## Key discovery must be cheap

Keys are computed during listing (`dagger check -l`, `dagger list`), for every
parent item. Whatever you run to produce keys, every user pays for on every
listing.

- **No containers.** Use `Workspace.findRoots`, `Workspace.file`,
  `Workspace.glob` and `Workspace.search` (ripgrep). Finding test names with
  `go test -list` needed the full test container per module; a search for
  `^func Test…\(\w+ \*testing\.T\)` needs none.
- **Don't route discovery through heavy helpers.** Keep "is this an item?"
  separate from "what does running it need?". Here, module validity and test
  names had been read from a compiled source scanner; answering them natively
  meant listing never builds or runs it. Prove it: make the helper fail on
  purpose, then list — it should still work.
- **Know what static discovery can't see.** A search doesn't evaluate build
  constraints, so it can list a test that won't compile in; selecting it runs
  nothing. Document it.

## Keys: uniqueness, order, failure

- **Unique.** Duplicate keys are rejected when the collection is read. Names
  repeat naturally (the same test name in two packages) — `.uniq`.
- **Deterministic order.** Keys feed cache keys and batch arguments (a `-run`
  pattern); an order that changes between runs busts caches. Search results
  arrive unordered. Dang has no list sort, but strings compare with `<` in byte
  order:

  ```dang
  let sorted(items: [String!]!): [String!]! {
    let empty: [String!]! = []
    items.reduce(empty) { acc, item =>
      acc.takeWhile { x => x <= item } + [item] + acc.dropWhile { x => x <= item }
    }
  }
  ```

- **Failure has no good home.** If computing keys raises, listing fails for the
  whole workspace. If you rescue to `[]`, the engine skips the batch for an
  empty selection and a broken item passes `dagger check` silently. Prefer key
  sources that still work when the code is broken (static search does), so the
  batch runs and fails loudly.

## Consumer cheat sheet

```console
$ dagger list go-modules                          # keys of a collection type
$ dagger list go-tests --go-module=sdk/go
$ dagger check -l --all --go -f=cli               # one line per key combination, as flags
$ dagger check --go --test --go-module=sdk/go --go-test=TestConnect
$ dagger check go/modules/tests/test --go-test=TestConnect    # a path works too
$ dagger check 'dag://go/modules/tests/test?go-module=sdk/go&go-test=TestConnect'
$ dagger -W ./sdk/go check                        # scope by directory instead
```

The selector flags:

| Flag | Selects |
|---|---|
| `--go`, `--by-go` | artifacts from module `go` |
| `--test`, `--check-test` | checks named `test` in **every** module (`--generator-<name>` for generate) |
| `--go-module=KEY` | one key of a dimension (repeatable) |
| `--go-modules` | the whole dimension |

- A short alias (`--go`, `--test`) exists only when the name means one thing;
  the `--by-`/`--check-` form always works.
- A key flag is the dimension's short name when that is unique among installed
  modules, otherwise the singular of its qualified name
  (`--golangci-lint-module`), and `--dimension-<identifier>` as a last resort.
  `dagger check --help` lists the flags in effect; the metavar comes from the
  `@keys` field (`paths` → `PATH`).
- `-f=cli` prints each listed line as flags you can paste back in.
- Name the **item** path (`go/modules/test`); the engine substitutes the batch.
  Dimension flags on the batch path (`go/modules/batch/test`) are rejected:
  no dimension crosses a batch path.
- Flags are resolved against the selected paths, so a flag for a dimension
  that isn't on the path is an unknown flag.
- `check -l` groups keys onto one line; add `--all` to see them.

## Gotchas

**Dimension names**
- The short name comes from the **item type** (`GoModule` → `go-module`), and
  authors can't override it. Generic item names collide across modules: with
  `go`, `golangci-lint` and `staticcheck` installed, the flags become
  `--go-module`, `--golangci-lint-module` and `--staticcheck-module`, and they
  change again whenever another module with the same item type is installed.
  Pick distinctive item type names if you can.

**Function names**
- A check's function name is its CLI flag (`--test`, `--check-test`) and its
  label in `dagger check -l`, and a batch has to share its item function's
  name. Pick a name that stands alone: `test` over `run`. Verbs that double as
  nouns (`test`, `lint`, `build`) still read as noun.verb on the type
  (`GoTest.test`).
- Sharing a name across modules is a feature: `--lint` selects every linter's
  `lint`, and `--golangci-lint --lint` narrows it.
- The `@get` function's name is never shown (consumers call `get(key:)`), but
  it lives in the same type as the batch, so it can't reuse the verb. Name it
  for the item: `function(name:)`, `module(path:)`. Its docstring **is** shown:
  `dagger list` uses it as each item's description.

**Workspace API paths are inconsistent**
- `Workspace.glob` patterns are always workspace-root-relative, and a leading
  `/` matches nothing.
- `Workspace.file`, `findRoots(start:)` and `search(paths:)` treat a leading
  `/` as the workspace root, and relative paths as cwd-relative.
- `findRoots` returns cwd-relative paths (`..` for an enclosing root) —
  convert before comparing.
- `search` returns `filePath` workspace-relative but sometimes `./`-prefixed
  (seen for a module at the root of a git workspace), while `glob` does not.
  Normalize both sides: `trimPrefix("/").trimPrefix("./")`.
- `search` with explicit **file** paths fails on a workspace built with
  `withDirectory` (`rg: a_test.go: No such file or directory`). Search the
  directory and filter the results to the files you want.
- Test synthetic workspaces **and** a real git workspace; the `./` prefix bug
  only showed on the latter.

**Dang**
- `str.match(...)` returns a nullable `Match`; anything read through it is
  nullable (`cannot use String as String!`). Use `?? ""` or plain string ops.
- `matchAll` gives `string`, `start`, `end`, `captures`.
- No list sort; `uniq` exists; strings compare with `<`; `toString(n)` for
  numbers.
- Backtick strings interpolate `${…}`; a single backtick is fine inside a
  triple-backtick string.
- You can't `map` a list of GraphQL objects directly — select first:
  `dirs.{{path, execute.{{pass}}}}`.
- Make back-references (`let module: GoModule!`) private. A public one is
  traversed and produces odd artifacts (`go/modules/tests/module`).

**Discovery side effects**
- Any module function returning a `Changeset` that needs no user input gets a
  `stale` check — including action helpers (dependency updates, version
  bumps). Installing a module that has them adds failing checks to every
  `dagger check`.

**Lockfiles**
- A module pinned to a branch (`github.com/org/mod@collections`) is locked to
  the commit the first run resolved. After pushing, `dagger update` (or remove
  the line from `dagger.lock`) or you'll keep testing old code.

**Tooling at the time of writing (dev builds)**
- Collections need `engineVersion: v1.0.0-beta.15`, which is unreleased. A
  migrated module fails to load on a released engine; run it on a dev
  engine, and don't take a released engine's error as a migration bug.
- `dagger call` can't navigate a collection yet (`typedef "[Item]" not found`).
  Test through a Dang dependency, `dagger list`, or `dagger check`.
- `check -l` doesn't say whether a batch will run; the grouped line shows the
  item's description.
- With `-m <module>`, `check -l -f=cli` prints module and dimension flags
  (`--go --go-module=…`) that the same command then rejects as unknown; only
  `--check-<name>`, `--by-<module>` and `dag://` addresses work there. Install
  the module in a workspace to use the flags.

## Verifying a migration

1. `dagger list <collection-type>` shows the keys you expect (install the
   module in a workspace `dagger.toml` if `-m` isn't supported by the command).
2. `dagger check -l` lists each check **once** — no aggregate alongside the
   item path, no parent check alongside a nested one.
3. Run a filtered check with `--progress=plain` and confirm one batch call and
   no item calls: count `_Batch.` spans against the item function's spans
   (e.g. `GoTests_Batch.test` vs `GoTest.test`).
4. Confirm the batch saw only the selected keys (the modules it built, the
   `-run` pattern it used).
5. In the test suite, cover: `get` in and outside a subset, `subset` order,
   unknown and duplicate keys rejected, empty subset valid, a subset batch
   running only its keys, and a delta-driven whole run versus a filtered run.
6. If discovery is supposed to avoid a helper, sabotage the helper and list.
7. Run every suite: a shared library change (discovery, source policy) reaches
   every tool built on it.
