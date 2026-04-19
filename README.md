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
- Map pipeline:
1. `loadmap.UseCase` orchestrates map extraction and conversion.
2. `MapReader` extracts raw `THINGS/LINEDEFS/SIDEDEFS/VERTEXES/SECTORS`.
3. `MapConverter` maps binary lumps to domain `Level` model.

Run:

```bash
go run ./cmd/doom -iwad /path/to/doom1.wad
```

Run headless runtime demo for 10 ticks:

```bash
go run ./cmd/doom -iwad /path/to/doom1.wad -runtime-ticks 10
```

Parse one map from a WAD:

```bash
go run ./cmd/doom -iwad /path/to/doom1.wad -map E1M1
```
