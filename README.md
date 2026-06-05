<div align="center">

# OpenCSV

**A fast, local-first CSV & Excel editor — a single self-contained binary.**

Open 100MB+ files in a virtual grid, sort/filter/query them like a spreadsheet,
run SQL against your data, and export anywhere. No cloud, no install, no telemetry.
An open-source alternative to [SmoothCSV](https://smoothcsv.com/).

</div>

---

## Why OpenCSV?

- **Local-first.** Your data never leaves your machine. The whole app is one Go binary that embeds the web UI.
- **Built for big files.** A virtual grid renders only what's on screen, and huge files switch to a windowed mode so browser memory stays bounded (a 1M-row / 90MB file keeps the tab at ~9MB of heap).
- **Spreadsheet ergonomics + SQL power.** Excel-style per-column sort/filter dropdowns *and* a SQL console that queries your CSV as a table.

## Features

- 📂 **Open** CSV / TSV / TXT / DAT (delimiter & encoding auto-detected: UTF-8/BOM, GBK, GB18030, UTF-16, Latin-1) and import/export **Excel (.xlsx)**
- ⚡ **Virtual grid** that stays smooth on millions of rows; inline cell editing, column rename, column resize
- 🔼 **Sort** — multi-key, type-aware (text / number / date / length); per-column buttons and a global dialog stay in sync
- 🔎 **Filter** — Excel-style per-column dropdown (filter by values with counts, or by condition) unified with a global builder and an editable `WHERE` bar. 20 operators incl. `contains`, `starts/ends with`, `like`, `between`, `is one of`, `is empty`, …
- 🔁 **Find & Replace** with regex, case sensitivity, and per-column scope
- 🧮 **SQL console** — query your data as a table named `data` (powered by an in-process SQLite engine, cached per session)
- ✏️ **Editing** — undo/redo, copy/paste (multi-cell, tab-separated), insert/delete rows & columns
- 🧰 **Transforms** — case conversion, trim, merge/split columns, deduplicate, transpose
- 📊 **Aggregate** stats on a selection (sum / avg / min / max / count / unique)
- 📤 **Export** — Excel, CSV, JSON, Markdown, HTML, SQL, LaTeX
- 🎨 Multi-tab UI, command palette (`⌘P`), light/dark/system theme, context menus, keyboard shortcuts

## Tech stack

| Layer | Tech |
|-------|------|
| Backend | Go 1.25 · [Gin](https://github.com/gin-gonic/gin) · [excelize](https://github.com/qax-os/excelize) · [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) (pure Go, **no CGO**) |
| Frontend | Vue 3 · TypeScript · Pinia · Vite · lucide-vue-next |
| Packaging | Frontend builds into `backend/dist/`, embedded via `//go:embed` → one standalone executable |

## Quick start

### Prerequisites
- Go **1.25+**
- Node **20+**

### Run in development (two terminals)

```bash
# Terminal 1 — backend API on :7070 (does not serve the UI in dev)
cd backend
DEV=1 go run .

# Terminal 2 — Vite dev server on :5173 (proxies /api and /ws to :7070)
cd frontend
npm install
npm run dev
```

Open <http://localhost:5173>.

### Build the production binary

```bash
cd frontend && npm run build      # outputs to ../backend/dist/
cd ../backend && go build -o opencsv .
./opencsv                         # serves UI + API on :7070, opens your browser
```

`PORT=8080 ./opencsv` overrides the port.

### Type-check the frontend

```bash
cd frontend && npx vue-tsc --noEmit
```

## Project structure

```
opencsv-cc/
├── backend/            # Go HTTP server + all business logic
│   ├── main.go         # Gin router, //go:embed dist, browser launch
│   ├── handlers/       # HTTP handlers (file, data, export, sql, upload)
│   ├── services/       # csv, excel, session, sql, transform, watcher
│   └── models/         # request/response + session models
├── frontend/           # Vue 3 SPA (builds into backend/dist/)
│   └── src/
│       ├── components/grid/VirtualGrid.vue   # core spreadsheet component
│       ├── components/dialogs|panels/        # sort/filter/find/sql UI
│       ├── stores/     # Pinia: tabs, settings, history
│       └── api/        # axios clients
├── scripts/gen_bench.py   # deterministic benchmark dataset generator
├── ARCHITECTURE.md        # detailed architecture notes (中文)
└── PRD.md                 # product requirements (中文)
```

> The architecture and PRD docs are currently written in Chinese — translations are very welcome (see Contributing).

## Performance

Measured on a 1,067,371-row, 90 MB real-world dataset (UCI *Online Retail II*):

| Operation | Time |
|-----------|------|
| Open file (90 MB) | ~280 ms |
| Filter (`Country is one of …`) | ~17 ms |
| Filter (`Description LIKE '%…%'`) | ~240 ms |
| Sort (date / number, 1M rows) | ~0.3 s |
| SQL query (repeat, cached table) | ~80 ms |
| Browser tab heap (windowed mode) | ~9 MB |

Need test data? `python3 scripts/gen_bench.py` regenerates benchmark CSVs deterministically (they're intentionally not committed).

## Contributing

Contributions are welcome — code, docs, translations, bug reports, and ideas.

1. **Fork & clone**, then get the dev servers running (see [Quick start](#quick-start)).
2. **Create a branch** off `main`: `git checkout -b feat/my-change`.
3. **Make your change.** Good first areas:
   - Adding a data operation: handler in `backend/handlers/data.go` → service in `backend/services/` → route in `backend/main.go` → API call in `frontend/src/api/data.ts` → wire it into the relevant component / command palette.
   - Adding an export format: `backend/handlers/export.go` + a route, then call it from the frontend.
   - Translating `ARCHITECTURE.md` / `PRD.md` to English.
4. **Check it builds:** `go build ./...` (backend) and `npx vue-tsc --noEmit` (frontend).
5. **Open a PR** with a clear description and, where it helps, a screenshot or before/after numbers.

Conventions:
- Frontend uses Vue 3 `<script setup lang="ts">`. No CSS frameworks — theming is via CSS custom properties in `frontend/src/style.css`.
- Backend stays **pure Go (no CGO)** — keep SQLite on `modernc.org/sqlite`.
- Commit messages follow a `type: summary` style (`feat:`, `fix:`, `perf:`, `chore:`, `docs:`).

See [`CLAUDE.md`](./CLAUDE.md) for a deeper guide to conventions and the codebase layout.

## License

Licensed under the [Apache License 2.0](./LICENSE). By contributing, you agree
that your contributions will be licensed under the same terms.
