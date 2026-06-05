# scripts/

Utility scripts for OpenCSV development.

## gen_bench.py — benchmark dataset generator

The performance numbers in `PRD.md` / project memory are measured against large
CSV files (tens to hundreds of MB). Those files are **not** committed to git;
this script regenerates them deterministically so the benchmarks can be
re-verified at any time.

```bash
# Generate all presets into /tmp (the default location)
python3 scripts/gen_bench.py

# Custom output directory
python3 scripts/gen_bench.py --out ./bench

# Only specific presets
python3 scripts/gen_bench.py m1-100mb 1m

# List available presets
python3 scripts/gen_bench.py --list
```

### Presets

| Name       | Rows × Cols      | Approx size | Purpose                                   |
|------------|------------------|-------------|-------------------------------------------|
| `m1-100mb` | 720,000 × 12     | ~108 MB     | "open a 100MB file" PRD target            |
| `1m`       | 1,300,000 × 3    | ~103 MB     | 1M+ row find / filter / sort benchmarks   |
| `window`   | 200,000 × 8      | ~19 MB      | virtual-scroll / window render benchmarks |
| `small`    | 5,000 × 3        | ~145 KB     | quick functional checks                   |

Output is deterministic (content is a fixed function of the row index), so
regenerating a preset produces a byte-identical file. The generated files match
the naming pattern `opencsv-*.csv`, which is gitignored.
