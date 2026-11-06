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
3. **`stitch.sh`** renders frames as GIF using Python + Pillow the way the docs site draws its terminal panels:
   JetBrains Mono from `docs/assets/fonts/`, the site's palette for the frames' ANSI colors, its line height and
   padding, at twice the CSS size so the GIFs stay sharp when the page scales them down

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
- **GIF rendering**: automatic via Python, Pillow and fontTools (installed by mise venv from `requirements.txt`)
- **Fonts**: DejaVu Sans Mono (`fonts-dejavu`) draws the few characters JetBrains Mono's site subsets lack, such as
  `✓` and `⚠`; without it they are left blank

`generate.sh` sets `CLICOLOR_FORCE=1`, since the recorder writes stdout to `/dev/null` and Lipgloss drops colors
without a terminal. Macro mode never runs terraform and can seed only a plan and a state, so the `output`,
`validate` and `workspace` tapes record empty screens, and a full run overwrites their GIFs with them. Their GIFs in
`docs/assets/demo/` are recorded with the binary built from commit `7d89c7a`, the last that accepts
`--outputs`, `--validate-result` and `--workspaces`, fed `demo/fixtures/outputs.json`, `validate.json` and
`workspaces.json` (issue #87).

## Docker demo

Try tfui without installing:

```bash
docker compose -f demo/docker-compose.yml up
# or
docker run --rm -it lmarqs/terraform-ui:demo
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

## Before/after on the landing page

The "See it work" tour at the top of the docs landing page plays one real plan: the raw `terraform plan` output,
tfui's tree, and the headless `tfui plan --ci` summary an agent reads. All of it comes from the fixture in
`demo/before-after/`, a few AWS resources in three modules with a hand-written `terraform.tfstate`.

```bash
mise run demo:before-after   # needs a logged-in claude CLI; rewrites docs/_includes/before-after/, docs/_data/before_after.yml and docs/_data/tour.json
```

`tour.json` holds the rows the tour's simulated plan view plays with: each change's address, action, the risk badge
tfui printed for it, and up to six attributes for the inspect screen. `docs/assets/tour.js` renders them.

Nothing reaches AWS. The provider has fake credentials and skips its account checks, and every plan runs with
`-refresh=false`, so terraform only diffs the config against the state file. tfui itself always plans with
`-refresh=true`, so the headless run goes through `terraform-offline`, a wrapper passed as `--terraform-bin` that
rewrites that flag. The first run downloads the AWS provider (about 700 MB).

To change the story, edit the modules or the state, rerun the task, and commit the regenerated files.
