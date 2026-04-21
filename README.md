# go-doom

Layered Doom engine rewrite in Go with strict separation:

- `internal/domain` - pure domain structures only.
- `internal/application` - services (use-cases).
- `*/dto.go` - DTO contracts are colocated with the layer/package that uses them.
- `internal/infrastructure` - binary parsers, adapters, converters.
- `cmd/doom` - bootstrap/CLI entrypoint.

## Current milestone

- WAD ingestion pipeline:
1. `BinaryReader` parses raw WAD binary into DTO.
2. `ArchiveConverter` maps DTO to domain `Archive`.
3. `loadiwad.Service` orchestrates reading and conversion.
- Runtime pipeline:
1. `rungame.Service` orchestrates tick/controls/simulation/render flow.
2. `CommandMapper` maps raw controls DTO into commands.
3. `DomainSimulation` + `HeadlessRenderer` provide infrastructure adapters.
- Map pipeline:
1. `loadmap.Service` orchestrates map extraction and conversion.
2. `MapReader` extracts raw `THINGS/LINEDEFS/SIDEDEFS/VERTEXES/SECTORS`.
3. `MapConverter` maps binary lumps to domain `Level` model.
- Playable window pipeline:
1. `playmap.Service` applies controls -> commands -> simulation each frame.
2. `ControlPoller` collects keyboard state.
3. `FirstPersonRenderer` renders walls in first-person perspective.
4. `TopDownRenderer` remains available via `-topdown` for debug.
5. `LevelSimulation` applies collision against blocking linedefs and spawn orientation.

## Bootstrap style

- `cmd/doom/main.go` runs `Load(ctx)` and then `app.Run(ctx, cfg.App, os.Stdout)`.
- `cmd/doom/config.go` contains startup config loading/validation.
- `internal/app/runner.go` contains runtime/bootstrap orchestration.

Config precedence (higher wins):

1. CLI flags
2. Environment variables (`DOOM_*`)
3. `-config` file (or `DOOM_CONFIG_PATH`)
4. `configs/default.json`
5. Code defaults

Supported env vars:

- `DOOM_CONFIG_PATH`
- `DOOM_IWAD_PATH`
- `DOOM_MAP`
- `DOOM_PLAY`
- `DOOM_TOPDOWN`
- `DOOM_WIDTH`
- `DOOM_HEIGHT`
- `DOOM_ZOOM`
- `DOOM_TICK_RATE`
- `DOOM_RUNTIME_TICKS`

Run:

```bash
go run ./cmd/doom -iwad /path/to/doom1.wad
```

Run with config file:

```bash
go run ./cmd/doom -config ./configs/dev.json
```

Run headless runtime demo for 10 ticks:

```bash
go run ./cmd/doom -iwad /path/to/doom1.wad -runtime-ticks 10
```

Parse one map from a WAD:

```bash
go run ./cmd/doom -iwad /path/to/doom1.wad -map E1M1
```

Run playable map window:

```bash
go run ./cmd/doom -iwad /path/to/freedoom1.wad -map E1M1 -play
```

Flags override config values. Example:

```bash
go run ./cmd/doom -config ./configs/dev.json -width 1600 -height 900
```

Controls:

- `W/S` or `ArrowUp/ArrowDown`: move forward/back
- `A/D` or `ArrowLeft/ArrowRight`: turn left/right
- `Q/E`: strafe left/right
- `Space`, `Ctrl`, or `LMB`: fire
- `Esc`: quit
