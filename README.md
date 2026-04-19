# go-doom

Layered Doom engine rewrite in Go with strict separation:

- `internal/domain` - pure domain entities and invariants.
- `internal/application` - use cases and DTO contracts.
- `internal/infrastructure` - binary parsers, adapters, converters.
- `cmd/doom` - bootstrap/CLI entrypoint.

## Current milestone

- WAD ingestion pipeline:
1. `BinaryReader` parses raw WAD binary into application DTO.
2. `WADConverter` maps DTO to domain `Archive`.
3. `loadiwad.UseCase` orchestrates reading and conversion.
- Runtime pipeline:
1. `rungame.UseCase` orchestrates tick/input/simulation/render flow.
2. `InputConverter` maps raw input DTO into domain commands.
3. `DomainSimulation` + `HeadlessRenderer` provide infrastructure adapters.

Run:

```bash
go run ./cmd/doom -iwad /path/to/doom1.wad
```

Run headless runtime demo for 10 ticks:

```bash
go run ./cmd/doom -iwad /path/to/doom1.wad -runtime-ticks 10
```
