# DevAtlas MVP

Minimal CLI to fetch Saramin jobs, normalize them, and write region counts.

## Requirements
- Go 1.21+
- Saramin API access key

## Run
```powershell
$env:SARAMIN_ACCESS_KEY="YOUR_KEY"
go run .\cmd\devatlas -job-cd 84,92 -updated-min 1700000000 -updated-max 1700086400
```

Output:
- `data/region_counts.json` (includes `meta.missing_regions`)
- `data/region_missing.jsonl` (missing region entries)
- `data/latest_companies.json` (current hiring companies)
- `data/geocode_cache.json` (address to coordinate cache)

Defaults:
- `job-cd` omitted: built-in developer job codes are used.
- `updated-min/max` omitted: last run window is used if `data/run_state.json` exists, otherwise last 24 hours.
- `current-days`: 21
- `refetch-days`: 7
- `min-interval-ms`: 200
- `retry-attempts`: 3
- `retry-base-ms`: 500
- `retry-max-ms`: 5000

## Schedule
```powershell
go run .\cmd\devatlas -schedule
```

Notes:
- Daily schedule runs at `00:10` local time by default (`-schedule-at HH:MM`).
- Successful runs update `data/run_state.json` (used for the next window).
- A refetch window runs on automatic schedules to recover missed updates (`-refetch-days`).

## GitHub Pages deploy
```powershell
$env:PAGES_REPO_URL="https://github.com/elecpapaya/devatlas.git"
$env:PAGES_BRANCH="gh-pages"
powershell -File .\scripts\deploy-pages.ps1
```
