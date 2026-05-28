# OpenCSV — 系统架构设计

**版本：** 1.0  
**日期：** 2026-05-27

---

## 1. 整体架构

OpenCSV 采用**本地 Web 应用架构**：

```
┌─────────────────────────────────────────────────────────┐
│                    用户浏览器                              │
│  ┌─────────────────────────────────────────────────┐    │
│  │            Vue 3 前端 (SPA)                      │    │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │    │
│  │  │ Grid 组件 │ │ 侧边工具栏 │ │  SQL Console     │ │    │
│  │  └──────────┘ └──────────┘ └──────────────────┘ │    │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │    │
│  │  │命令面板   │ │ 查找替换  │ │  多标签页管理     │ │    │
│  │  └──────────┘ └──────────┘ └──────────────────┘ │    │
│  └─────────────────────────────────────────────────┘    │
│                      HTTP/WebSocket                       │
└───────────────────────────┬─────────────────────────────┘
                             │
┌───────────────────────────▼─────────────────────────────┐
│              Go 本地服务器 (localhost:7070)               │
│  ┌──────────────────────────────────────────────────┐   │
│  │  Gin Router                                       │   │
│  │  ┌─────────────┐  ┌──────────────┐               │   │
│  │  │ File Handler │  │ Data Handler │               │   │
│  │  └─────────────┘  └──────────────┘               │   │
│  │  ┌─────────────┐  ┌──────────────┐               │   │
│  │  │ SQL Handler  │  │ Export Handler│              │   │
│  │  └─────────────┘  └──────────────┘               │   │
│  └──────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────┐   │
│  │  服务层 (Service Layer)                            │   │
│  │  ┌──────────┐ ┌──────────┐ ┌───────────────────┐ │   │
│  │  │CSV Parser│ │SQLite Eng│ │ ExcelIO (excelize) │ │   │
│  │  └──────────┘ └──────────┘ └───────────────────┘ │   │
│  │  ┌──────────┐ ┌──────────┐                        │   │
│  │  │FileWatcher│ │ Transform│                        │   │
│  │  └──────────┘ └──────────┘                        │   │
│  └──────────────────────────────────────────────────┘   │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │  静态文件服务（嵌入 Vue 构建产物）                    │   │
│  └──────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────┘
                             │
                    ┌────────▼────────┐
                    │  本地文件系统     │
                    │  (CSV/XLSX 文件) │
                    └─────────────────┘
```

---

## 2. 后端架构（Go）

### 2.1 目录结构

```
backend/
├── main.go                  # 程序入口，启动 Gin 服务器
├── go.mod
├── go.sum
├── config/
│   └── config.go            # 配置管理
├── handlers/
│   ├── file.go              # 文件操作 API
│   ├── data.go              # 数据操作 API
│   ├── sql.go               # SQL 查询 API
│   ├── export.go            # 导入/导出 API
│   └── settings.go          # 设置 API
├── services/
│   ├── csv/
│   │   ├── parser.go        # CSV 解析（流式）
│   │   ├── writer.go        # CSV 写入
│   │   └── detector.go      # 编码/分隔符自动检测
│   ├── sql/
│   │   └── engine.go        # SQLite 内存引擎
│   ├── excel/
│   │   └── excel.go         # Excel 导入/导出
│   ├── transform/
│   │   └── transform.go     # 文本转换
│   ├── watcher/
│   │   └── watcher.go       # 文件变化监听
│   └── session/
│       └── session.go       # 文件会话管理（内存缓存）
├── models/
│   ├── csv.go               # CSV 数据模型
│   ├── request.go           # 请求/响应模型
│   └── settings.go          # 设置模型
└── middleware/
    ├── cors.go
    └── logger.go
```

### 2.2 数据流：打开大文件

```
客户端请求 → File Handler
    → CSV Parser (流式读取，分块)
    → Session Store (内存 + mmap 可选)
    → 返回首屏数据 (前 1000 行) + 总行数
客户端滚动 → 请求指定行范围
    → Session 直接返回缓存数据
```

### 2.3 核心依赖

| 包 | 用途 |
|----|------|
| `github.com/gin-gonic/gin` | HTTP 路由框架 |
| `github.com/xuri/excelize/v2` | Excel 读写 |
| `github.com/mattn/go-sqlite3` | SQLite 引擎（CGO） |
| `github.com/fsnotify/fsnotify` | 文件变化监听 |
| `golang.org/x/text/encoding` | 多编码支持 |
| `github.com/gorilla/websocket` | WebSocket（文件变化通知）|

---

## 3. 前端架构（Vue 3）

### 3.1 目录结构

```
frontend/
├── index.html
├── vite.config.ts
├── package.json
├── tsconfig.json
├── src/
│   ├── main.ts
│   ├── App.vue
│   ├── router/
│   │   └── index.ts
│   ├── stores/                   # Pinia 状态管理
│   │   ├── tabs.ts               # 多标签页状态
│   │   ├── grid.ts               # 网格数据状态
│   │   ├── settings.ts           # 用户设置
│   │   └── history.ts            # 撤销/重做历史
│   ├── components/
│   │   ├── layout/
│   │   │   ├── AppLayout.vue     # 主布局
│   │   │   ├── TabBar.vue        # 多标签页栏
│   │   │   ├── Toolbar.vue       # 顶部工具栏
│   │   │   └── StatusBar.vue     # 底部状态栏
│   │   ├── grid/
│   │   │   ├── VirtualGrid.vue   # 虚拟化表格核心
│   │   │   ├── GridCell.vue      # 单元格组件
│   │   │   ├── GridHeader.vue    # 列头组件
│   │   │   └── CellEditor.vue    # 单元格编辑器
│   │   ├── dialogs/
│   │   │   ├── OpenFileDialog.vue
│   │   │   ├── FindReplaceDialog.vue
│   │   │   ├── FilterDialog.vue
│   │   │   ├── SortDialog.vue
│   │   │   ├── ImportDialog.vue
│   │   │   └── ColumnStatsDialog.vue
│   │   ├── panels/
│   │   │   ├── SqlConsole.vue    # SQL 控制台侧面板
│   │   │   └── AggregatePanel.vue
│   │   └── CommandPalette.vue    # 命令面板
│   ├── composables/
│   │   ├── useGrid.ts            # 网格逻辑
│   │   ├── useKeyboard.ts        # 键盘快捷键
│   │   ├── useSelection.ts       # 选区管理
│   │   ├── useHistory.ts         # 撤销/重做
│   │   └── useFileWatch.ts       # 文件监听 WebSocket
│   ├── api/
│   │   ├── client.ts             # Axios 实例
│   │   ├── file.ts               # 文件相关 API
│   │   ├── data.ts               # 数据操作 API
│   │   └── sql.ts                # SQL API
│   ├── types/
│   │   └── index.ts              # TypeScript 类型定义
│   └── utils/
│       ├── clipboard.ts          # 剪贴板工具
│       ├── format.ts             # 多格式转换
│       └── shortcuts.ts          # 快捷键注册
```

### 3.2 状态管理（Pinia）

```typescript
// tabs store
{
  tabs: Tab[]           // 所有打开的文件标签
  activeTabId: string   // 当前活动标签
}

// grid store (per tab)
{
  fileId: string        // 文件会话 ID
  columns: Column[]     // 列定义
  totalRows: number     // 总行数
  cache: Map<number, Row[]>  // 行缓存（按窗口）
  selection: Selection  // 当前选区
  editingCell: Cell | null   // 当前编辑中的格
}
```

### 3.3 虚拟网格原理

```
可视区域高度 = 600px
行高 = 28px
可见行数 = Math.ceil(600/28) + 缓冲 = ~30 行

滚动位置 → 计算起始行 = Math.floor(scrollTop/28)
          → 从缓存或 API 获取 [startRow, startRow+50] 的数据
          → 只渲染这 ~50 行 DOM
          → 用 padding-top 撑起滚动高度
```

---

## 4. API 设计

### 4.1 文件 API

```
POST   /api/files/open           打开文件，返回 fileId + 首屏数据
GET    /api/files/:id/rows       分页获取行数据 ?offset=0&limit=100
PUT    /api/files/:id/cells      批量更新单元格
POST   /api/files/:id/save       保存到原路径
POST   /api/files/:id/save-as    另存为
DELETE /api/files/:id            关闭并释放会话
GET    /api/files/recent         最近文件列表
```

### 4.2 数据操作 API

```
POST   /api/files/:id/sort       排序
POST   /api/files/:id/filter     筛选
POST   /api/files/:id/find       查找，返回匹配位置列表
POST   /api/files/:id/replace    替换
POST   /api/files/:id/rows/insert    插入行
DELETE /api/files/:id/rows           删除行
POST   /api/files/:id/columns/insert  插入列
DELETE /api/files/:id/columns         删除列
POST   /api/files/:id/transform       文本转换
POST   /api/files/:id/deduplicate     去重
POST   /api/files/:id/aggregate       统计
```

### 4.3 SQL & 导入导出 API

```
POST   /api/files/:id/sql        执行 SQL 查询
POST   /api/import/excel         导入 Excel
POST   /api/files/:id/export/excel    导出 Excel
POST   /api/files/:id/export/format   复制为指定格式（json/markdown/html/sql）
```

### 4.4 WebSocket

```
WS /ws/files/:id/watch    文件变化通知
```

---

## 5. 性能策略

### 5.1 大文件处理

1. **流式解析**：Go 后端用 `bufio.Scanner` 流式读取，不把整个文件加载进内存
2. **分块存储**：每 1000 行一个 chunk，存储在内存 map 中
3. **LRU 缓存**：超过内存限制时，淘汰最久未访问的 chunk，按需重新读取
4. **前端虚拟滚动**：只渲染可见 DOM，滚动时滑窗请求数据
5. **预取**：向上/下提前 2 个窗口预取数据

### 5.2 编辑性能

1. **本地优先**：编辑操作先更新前端状态，异步同步到后端
2. **批量写入**：收集 500ms 内的所有 cell 更新，批量发一次 API
3. **增量保存**：保存时只写变更的行（diff）

---

## 6. 技术选型汇总

| 层 | 技术 | 版本 |
|----|------|------|
| 后端框架 | Go + Gin | Go 1.22+ |
| 前端框架 | Vue 3 + TypeScript | Vue 3.4+ |
| 构建工具 | Vite | 5.x |
| 状态管理 | Pinia | 2.x |
| HTTP 客户端 | Axios | 1.x |
| 虚拟滚动 | @tanstack/vue-virtual | 3.x |
| Excel 处理 | excelize/v2 (Go) | 2.x |
| SQL 引擎 | go-sqlite3 (CGO) | 1.x |
| 文件监听 | fsnotify | 1.x |
| CSS 框架 | 自定义 CSS + CSS Variables | — |
| 图标 | Lucide Vue | 0.4x |
