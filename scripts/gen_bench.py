#!/usr/bin/env python3
"""
Generate the benchmark CSV datasets used for OpenCSV performance testing.

The datasets themselves are large (tens to hundreds of MB) and are intentionally
NOT committed to git. This script reproduces them deterministically so the
performance numbers in the PRD / project memory can be re-verified at any time.

Usage:
    python3 scripts/gen_bench.py                 # generate all presets into /tmp
    python3 scripts/gen_bench.py --out ./bench    # custom output directory
    python3 scripts/gen_bench.py m1-100mb 1m      # only the named presets
    python3 scripts/gen_bench.py --list           # list presets and exit

Presets (name -> rows x cols, approx size):
    m1-100mb   720,000 x 12   ~108 MB   "open a 100MB file" PRD target
    1m       1,300,000 x  3   ~103 MB   1M+ row find / filter / sort
    window     200,000 x  8    ~19 MB   virtual-scroll / window render
    small         5,000 x  3    ~145 KB quick functional checks

All output is deterministic (fixed content per row index), so regenerating
produces byte-identical files.
"""
import argparse
import os
import sys

# Default output directory matches where the ad-hoc benchmark files lived.
DEFAULT_OUT = "/tmp"


def gen_m1_100mb(f, rows: int = 720_000):
    """Wide-ish rows (12 columns) mixing ints, dates, text and decimals."""
    f.write("id,field_01,field_02,field_03,field_04,field_05,"
            "field_06,field_07,field_08,field_09,field_10,field_11\n")
    for i in range(1, rows + 1):
        seg = f"segment_{i % 8}"
        dec = f"{(i % 100) + (i % 7) / 7:.2f}"
        f.write(
            f"{i},{seg},2026-05-01,note row {i} col 4,{(i % 100):.2f},"
            f"{seg},2026-05-01,note row {i} col 8,{dec},"
            f"{seg},2026-05-01,note row {i} col 12\n"
        )


def gen_1m(f, rows: int = 1_300_000):
    """Narrow 3-column table; the `note` column is a zero-padded counter."""
    f.write("id,name,note\n")
    for i in range(1, rows + 1):
        f.write(f"{i},Name{i},{i:064d}\n")


def gen_window(f, rows: int = 200_000):
    """8 columns, moderate size, for scroll/window render benchmarks."""
    f.write("id,country,city,segment,amount,date,gender,note\n")
    countries = ["China", "Togo", "Argentina", "Australia", "Austria",
                 "Albania", "Algeria", "Belgium"]
    cities = ["Shanghai", "Beijing", "Sydney", "Vienna", "Tirana"]
    for i in range(1, rows + 1):
        f.write(
            f"{i},{countries[i % len(countries)]},{cities[i % len(cities)]},"
            f"segment_{i % 5},{(i % 1000) + 0.5:.2f},2026-05-01,"
            f"{'Male' if i % 2 else 'Female'},note {i}\n"
        )


def gen_small(f, rows: int = 5_000):
    """Tiny dataset with repeated category values for filter/dedupe checks."""
    f.write("id,country,city\n")
    countries = ["Afghanistan", "Albania", "Algeria", "Argentina",
                 "Australia", "Austria", "Azerbaijan", "Bangladesh",
                 "Belarus", "Belgium"]
    for i in range(1, rows + 1):
        c = countries[i % len(countries)]
        f.write(f"{i},{c},{c}-city\n")


PRESETS = {
    "m1-100mb": ("opencsv-m1-100mb.csv", gen_m1_100mb),
    "1m":       ("opencsv-1m.csv",        gen_1m),
    "window":   ("opencsv-bench-window.csv", gen_window),
    "small":    ("opencsv-small.csv",     gen_small),
}


def main(argv):
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("presets", nargs="*", help="preset names to generate (default: all)")
    ap.add_argument("--out", default=DEFAULT_OUT, help=f"output directory (default: {DEFAULT_OUT})")
    ap.add_argument("--list", action="store_true", help="list presets and exit")
    args = ap.parse_args(argv)

    if args.list:
        for name, (fname, _) in PRESETS.items():
            print(f"{name:10s} -> {fname}")
        return 0

    names = args.presets or list(PRESETS.keys())
    unknown = [n for n in names if n not in PRESETS]
    if unknown:
        print(f"Unknown preset(s): {', '.join(unknown)}", file=sys.stderr)
        print(f"Available: {', '.join(PRESETS)}", file=sys.stderr)
        return 2

    os.makedirs(args.out, exist_ok=True)
    for name in names:
        fname, fn = PRESETS[name]
        path = os.path.join(args.out, fname)
        print(f"Generating {name} -> {path} ...", flush=True)
        with open(path, "w", newline="") as f:
            fn(f)
        size_mb = os.path.getsize(path) / (1024 * 1024)
        print(f"  done ({size_mb:.1f} MB)")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
