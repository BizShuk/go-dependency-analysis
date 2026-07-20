# Policy examples

These policies illustrate dependency directions for common Go architectures. They are examples, not defaults. Replace `example.com/project` and package prefixes with the module paths used by the analyzed workspace.

## Standard layout

`standard-layout.json` prevents domain and service packages from depending on command entry points.

```bash
go-dependency-analysis --workspace /path/to/go.work --policy examples/standard-layout.json --fail-on error
```

## Clean Architecture

`clean-architecture.json` keeps entities independent from use cases and adapters, and keeps use cases independent from adapters. Dependencies point inward.

```bash
go-dependency-analysis --workspace /path/to/go.work --policy examples/clean-architecture.json --fail-on error
```

## Hexagonal Architecture

`hexagonal-architecture.json` keeps domain and application ports independent from adapters and prevents inbound and outbound adapters from depending on each other.

```bash
go-dependency-analysis --workspace /path/to/go.work --policy examples/hexagonal-architecture.json --fail-on error
```

Policy matching uses exact paths or a trailing `*` prefix match. `forbidden` rules apply to both package and module edges, so package-oriented examples work within a single Go module while module-oriented policies remain supported.
