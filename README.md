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
- `updated-min/max` omitted: last 24 hours window is used.
- `current-days`: 21
- `min-interval-ms`: 200
- `retry-attempts`: 3
- `retry-base-ms`: 500
- `retry-max-ms`: 5000

## GitHub Actions
This repo runs collection and deployment in GitHub Actions.

Required secret:
- `SARAMIN_ACCESS_KEY`

Workflow:
- `.github/workflows/collect.yml`
  - Runs daily at 00:10 KST (cron 10 15 * * *).
