# Demo

Generate animated GIFs from macro tapes for the documentation site.

## Quick start

```bash
# 1. Build the binary
mise run build

# 2. Generate recordings + stitch GIFs (all-in-one)
mise run demo:generate
```

## How it works

1. **Tapes** (`demo/tapes/*.tape`) define scripted interactions using the macro DSL
2. **`--record`** flag captures each frame as ANSI text with timing metadata
3. **`stitch.sh`** renders frames as GIF using Python + Pillow

```
tape + fixtures → tfui --macro --record → frames/ + manifest.json → stitch.sh → .gif
```

## Recording your own sessions

```bash
# Record an interactive session
tfui --plan ./demo/fixtures/plan-large.json --state ./demo/fixtures/state-large.json --record ./my-session/

# After quitting, you get:
ls my-session/
# manifest.json  recording.tape  frame_0001.txt  frame_0002.txt  ...

# The tape is replayable
tfui --plan ./demo/fixtures/plan-large.json --macro ./my-session/recording.tape

# Convert to GIF
./demo/stitch.sh ./my-session/ my-session.gif
```

## Prerequisites

- **Build**: `mise run build` (Go 1.25+)
- **GIF rendering**: automatic via Python + Pillow (installed by mise venv from `requirements.txt`)

## Docker demo

Try tfui without installing:

```bash
docker compose -f demo/docker-compose.yml up
# or
docker run -it ghcr.io/lmarqs/terraform-ui:demo
```

## File structure

```
demo/
├── fixtures/          # Realistic plan + state for demos
│   ├── plan-large.json
│   └── state-large.json
├── tapes/             # Macro scripts (one per feature)
│   ├── hero.tape
│   ├── plan-review.tape
│   ├── state-browse.tape
│   ├── pin-target.tape
│   ├── risk-analysis.tape
│   └── phantom.tape
├── output/            # Generated recordings + GIFs (gitignored)
├── generate.sh        # Full pipeline script
├── stitch.sh          # Frames → GIF converter
├── Dockerfile         # Demo container
└── docker-compose.yml
```
