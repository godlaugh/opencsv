# CLAUDE.md — OpenCSV

OpenCSV is a local-first CSV/Excel editor built as a single Go binary that embeds a Vue 3 SPA. It targets large files (100MB+) with virtual grid rendering, SQL queries, multi-format export, and full undo/redo.

---

## Repository Layout

```
opencsv-cc/
├── backend/          # Go HTTP server + all business logic
├── frontend/         # Vue 3 SPA (TypeScript, Pinia, Vite)
├── ARCHITECTURE.md   # Detailed architecture notes (Chinese)
├── PRD.md            # Product requirements document
└── .claude/          # Claude Code launch config
```

The frontend **builds into `backend/dist/`**, which the Go binary embeds via `//go:embed dist`. The result is a single self-contained executable.

---

## Development Workflow

### Prerequisites
- Go 1.25+
- Node 20+

### Run in development (two terminals)

**Terminal 1 — Go backend:**
```bash
cd backend
DEV=1 go run .
# Listens on localhost:7070; does NOT serve frontend files
```

**Terminal 2 — Vite dev server:**
```bash
cd frontend
npm install
npm run dev
# Runs on localhost:5173; proxies /api/* and /ws/* → localhost:7070
```

Open `http://localhost:5173` in the browser.

### Build production binary

```bash
cd frontend
npm run build          # Outputs to ../backend/dist/

cd ../backend
go build -o opencsv .  # Embeds dist/, produces standalone binary
./opencsv              # Starts server on :7070, auto-opens browser
```

`PORT` env var overrides the default port (`PORT=8080 ./opencsv`).

### Type-check frontend only

```bash
cd frontend
npx vue-tsc --noEmit
```

---

## Backend (Go)

### Package structure

```
backend/
├── main.go              # Gin router, embed directive, browser launch
├── middleware/
│   └── cors.go          # CORS for localhost:5173 and :7070
├── handlers/
│   ├── file.go          # File open/save/close/info/recent
│   ├── data.go          # Sort, filter, find/replace, insert/delete rows/cols,
│   │                    # transform, deduplicate, aggregate, transpose
│   ├── export.go        # Excel export, multi-format (JSON/Markdown/HTML/SQL)
│   ├── sql.go           # SQL SELECT execution
│   └── upload.go        # File upload endpoint
├── services/
│   ├── csv/
│   │   ├── parser.go    # Streaming CSV parse with encoding support
│   │   └── detector.go  # Auto-detect delimiter and encoding
│   ├── excel/
│   │   └── excel.go     # XLSX read/write via excelize/v2
│   ├── session/
│   │   └── session.go   # In-memory session store (Global singleton, RWMutex)
│   ├── sql/
│   │   └── engine.go    # SQLite in-process query engine
│   ├── transform/
│   │   └── transform.go # Case conversion, trim, merge/split columns
│   └── watcher/
│       └── watcher.go   # fsnotify file-change → WebSocket push
└── models/
    ├── csv.go           # CsvConfig, FileSession, Column, Cell, Sort/Filter models
    └── request.go       # Request/response DTOs
```

### Session model

All file data lives in `session.Global` — a mutex-protected map of `FileSession` values.

```go
type FileSession struct {
    ID        string
    FilePath  string
    Config    CsvConfig
    Columns   []Column
    Rows      [][]string   // full in-memory dataset
    TotalRows int
    Modified  bool
}
```

All data is loaded into `[][]string` at open time. There is no lazy-load chunking in the current implementation — the architecture doc describes aspirational chunking, but the shipped code reads the full file.

### Handler conventions

- Each handler is `func(c *gin.Context)` registered on the Gin router.
- Errors return JSON `{"error": "..."}` with the appropriate HTTP status (400/404/500).
- Success responses use `gin.H{}` maps or typed structs via `c.JSON(200, ...)`.
- Session IDs are UUID strings generated at open time; all per-file routes use `/:id`.

### Supported file formats (open)

CSV, TSV, TXT, DAT — delimiter auto-detected. Encodings: UTF-8, UTF-8 BOM, GBK, GB18030, UTF-16 LE/BE, Latin-1.

XLSX — imported via excelize; converted to in-memory `[][]string`.

### WebSocket

`GET /ws/files/:id/watch` streams file-change events from fsnotify to the frontend using Gorilla WebSocket.

---

## Frontend (Vue 3 + TypeScript)

### Directory structure

```
frontend/src/
├── main.ts              # App init, Pinia setup
├── App.vue              # Root: global keyboard shortcuts, notification toasts
├── style.css            # CSS custom properties (design tokens)
├── router/index.ts      # Vue Router (currently single-page, no real routes)
├── stores/
│   ├── tabs.ts          # Multi-tab state: open files, activeTabId
│   ├── settings.ts      # User prefs: theme (dark/light/auto)
│   └── history.ts       # Undo/redo entries
├── components/
│   ├── layout/
│   │   ├── AppLayout.vue
│   │   ├── TabBar.vue
│   │   ├── Toolbar.vue
│   │   ├── StatusBar.vue
│   │   └── WelcomeScreen.vue
│   ├── grid/
│   │   └── VirtualGrid.vue   # Core spreadsheet component (~1500 LOC)
│   ├── dialogs/
│   │   ├── FindReplaceBar.vue
│   │   ├── SortDialog.vue
│   │   ├── FilterDialog.vue
│   │   └── TransformMenu.vue
│   ├── panels/
│   │   └── SqlConsole.vue
│   ├── CommandPalette.vue
│   └── ContextMenu.vue
├── api/
│   ├── client.ts        # Axios instance, base URL /api
│   ├── file.ts          # File open/save/close/rows/cells/columns APIs
│   └── data.ts          # Sort/filter/find/replace/transform/aggregate APIs
├── types/index.ts       # Shared TypeScript interfaces
├── composables/         # Vue composables (useGrid, useKeyboard, useSelection, etc.)
└── utils/               # clipboard.ts, format.ts, shortcuts.ts
```

### State management (Pinia)

**`useTabsStore`** — central store for all open files:
- `tabs: Tab[]` — each `Tab` holds the `FileSession`, `rows` (page cache as `Map<number, string[][]>`), selection state, filter state
- `activeTabId: string`
- Key actions: `addTab`, `removeTab`, `setActiveTab`, `updateTabRows`, `markModified`

**`useSettingsStore`** — theme preference, persisted to localStorage.

**`useHistoryStore`** — undo/redo stack per tab. `HistoryEntry` stores `before`/`after` cell arrays.

### VirtualGrid rendering

Row height: **28px**. The grid renders only ~30 visible rows + an overflow buffer. `padding-top` is used to simulate scroll position without rendering off-screen DOM.

Page cache uses windows of 200 rows keyed by page index. On scroll, the composable fetches adjacent pages from the API and populates the cache.

### Styling conventions

No CSS framework. All theming via CSS custom properties defined in `style.css`:
- Colors: `--bg-app`, `--bg-panel`, `--bg-hover`, `--text-primary`, `--text-secondary`, `--border`, `--accent`
- Spacing: `--radius`, `--shadow-sm`, `--shadow-md`

Components use `<style scoped>` referencing these variables. Do not introduce Tailwind or other utility-class frameworks.

### Component conventions

- All components use Vue 3 `<script setup lang="ts">` syntax.
- Emit events for parent-driven state changes; use the store for cross-component state.
- Keyboard shortcuts live in `App.vue` (global) or within the component that owns the context (e.g., grid navigation in `VirtualGrid.vue`).

---

## API Reference

### File management

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/files/open` | Open file by path; returns `FileSession` + first 200 rows |
| GET | `/api/files/:id/rows` | Paginated rows (`?offset=&limit=`) |
| PUT | `/api/files/:id/cells` | Batch update cells |
| PUT | `/api/files/:id/columns` | Rename columns |
| POST | `/api/files/:id/save` | Save to original path |
| DELETE | `/api/files/:id` | Close session, free memory |
| GET | `/api/files/:id` | Get file info |
| GET | `/api/files/recent` | Recent files list |
| POST | `/api/files/upload` | Upload file bytes |

### Data operations

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/files/:id/sort` | Sort by one or more keys |
| POST | `/api/files/:id/filter` | Filter with AND/OR conditions |
| POST | `/api/files/:id/find` | Find (regex, case options); returns match positions |
| POST | `/api/files/:id/replace` | Replace matches |
| POST | `/api/files/:id/rows/insert` | Insert blank rows |
| DELETE | `/api/files/:id/rows` | Delete rows by index array |
| POST | `/api/files/:id/columns/insert` | Insert columns |
| DELETE | `/api/files/:id/columns` | Delete columns by index array |
| POST | `/api/files/:id/transform` | Text transforms (case, trim, merge/split) |
| POST | `/api/files/:id/deduplicate` | Remove duplicate rows |
| POST | `/api/files/:id/aggregate` | Statistics on selection (sum/avg/min/max/count/unique) |
| POST | `/api/files/:id/transpose` | Swap rows and columns |

### SQL & export

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/files/:id/sql` | Execute SQL SELECT via SQLite |
| POST | `/api/files/import/excel` | Import XLSX |
| POST | `/api/files/:id/export/excel` | Export to XLSX |
| POST | `/api/files/:id/export/format` | Export as JSON / Markdown / HTML / SQL |

---

## Key Conventions for AI Assistants

### Adding a new data operation

1. **Backend:** Add a handler in `handlers/data.go`, add service logic in the appropriate `services/` package, register the route in `main.go`.
2. **Frontend:** Add the API call in `api/data.ts`, wire it up in the relevant composable or dialog component, expose it via the command palette in `CommandPalette.vue`.

### Adding a new export format

Handler goes in `handlers/export.go`. Register the route in `main.go` under the `file` group. The frontend calls it from `api/data.ts` or a new `api/export.ts`.

### Modifying the grid

`VirtualGrid.vue` is large and owns selection, editing, context menu, keyboard navigation, and virtual scroll. Read it fully before editing. State that must survive tab-switch belongs in `useTabsStore`, not component-local `ref`.

### Session safety

All backend handlers must acquire a session via `session.Global.Get(id)` and return 404 if not found. Mutating handlers should use `session.Global.Set(id, updated)` after changes. The store's `RWMutex` is for concurrent HTTP requests — handlers themselves are not goroutine-safe.

### Do not

- Add CSS frameworks (Tailwind, Bootstrap, etc.) — use CSS custom properties.
- Add a separate database for metadata — session state is in-memory only.
- Introduce React or other frontend frameworks.
- Use CGO dependencies — the current SQLite is `modernc.org/sqlite` (pure Go). Do not swap to `mattn/go-sqlite3`.
- Auto-open the browser during development (`DEV=1` suppresses it).
