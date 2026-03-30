# Dexter

A fast Elixir module and function index for go-to-definition. Built as a "poor-man's LSP" for large Elixir codebases where traditional LSP servers are too slow or unavailable.

Dexter parses `.ex` and `.exs` files, extracts `defmodule`, `def`, `defp`, `defmacro`, and `defmacrop` definitions, and stores them in a local SQLite database. Lookups are instant (~10ms) regardless of codebase size.

## Why?

Elixir LSP servers (ElixirLS, Lexical, etc.) can struggle with very large umbrella apps. Ctags works but doesn't understand Elixir module namespacing, so `Foo` often resolves to the wrong module. Dexter sits in between — it's Elixir-aware (understands modules, nested modules, heredocs) but doesn't try to be a full LSP. Just fast, correct go-to-definition.

## Install

Requires Go 1.21+ and CGo (for SQLite).

```sh
git clone gitlab.com/remote-com/employ-starbase/dexter.git
cd dexter
go build -o dexter ./cmd/
# Move the binary somewhere on your PATH, or reference it directly
cp dexter /usr/local/bin/
```

## Usage

### Index a project

```sh
# First time — indexes all .ex/.exs files (including deps/)
dexter init ~/code/my-elixir-project

# Re-init from scratch (deletes existing index)
dexter init --force ~/code/my-elixir-project
```

This creates a `.dexter.db` file at the project root. Add it to your `.gitignore`.

### Look up definitions

```sh
# Find where a module is defined
dexter lookup MyApp.Repo
# => /path/to/lib/my_app/repo.ex:1

# Find where a function is defined
dexter lookup MyApp.Repo get
# => /path/to/lib/my_app/repo.ex:15

# Multiple function heads are returned as separate results
dexter lookup MyApp.Handlers.Webhooks process_event
# => /path/to/lib/my_app/handlers/webhooks.ex:20
# => /path/to/lib/my_app/handlers/webhooks.ex:28
```

### Keep the index up to date

```sh
# Re-index a single file (fast — ~10ms)
dexter reindex /path/to/lib/my_app/repo.ex

# Re-index the whole project (only re-parses files with changed mtimes)
dexter reindex ~/code/my-elixir-project

# Watch for file changes and re-index automatically
dexter watch ~/code/my-elixir-project
```

## Neovim Integration

Dexter is designed to be called from Neovim for go-to-definition. The Lua integration handles:

- **Alias resolution** — `alias MyApp.Handlers.Foo` and `alias MyApp.Handlers.Foo, as: Cool`
- **Multi-alias** — `alias MyApp.Handlers.{Foo, Bar}`
- **Import resolution** — `import MyApp.Helpers.Formatting` for bare function calls
- **Current buffer lookup** — private function calls resolve without leaving the buffer
- **Jumplist support** — `Ctrl-o` works to jump back
- **Telescope picker** — when multiple files define the same module, a picker is shown

See `lua/helpers/elixir-goto.lua` in the nvim config for the full integration.

### Automatic reindexing on save

The Neovim integration includes a `BufWritePost` autocmd that reindexes the current file whenever you save an `.ex` or `.exs` file. This keeps the index current without manual intervention.

## How it works

1. **Parsing** — Each `.ex`/`.exs` file is scanned line-by-line for `defmodule`, `def`, `defp`, `defmacro`, and `defmacrop` declarations. The parser tracks module nesting via `end` keywords and skips heredoc blocks (`"""`) to avoid indexing code examples in documentation.

2. **Storage** — Definitions are stored in a SQLite database (`.dexter.db`) with indexes on module name and module+function for fast lookups.

3. **Parallel indexing** — Initial indexing uses all available CPU cores for parsing, with a single writer for SQLite. On a 57k-file codebase, full indexing takes ~8 seconds.

4. **Incremental updates** — File mtimes are tracked. `reindex` only re-parses files that have changed since the last index.

## Performance

Measured on a 57k-file Elixir umbrella app (2.5M lines, 330k+ definitions):

| Operation | Time |
|-----------|------|
| Full init | ~8s |
| Lookup | ~10ms |
| Single file reindex | ~10ms |
| Full reindex (no changes) | ~2s |
