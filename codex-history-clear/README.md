# Codex History Manager

> 本文按 2026-07-03 的仓库代码和本地验证结果整理。

`codex-history-manager` 是一个面向 Windows 的本地桌面工具，用来查看和治理 Codex 会话历史。技术栈是 Go + Wails 2 + React/TypeScript。它现在的重点不是“清缓存”，而是把本地会话历史的扫描、计划、备份、删除、回滚和证据导出做成一条可复核的流程。

## 当前真实状态

- 这已经不是 Wails 默认模板，桌面壳、前端界面和后端服务都已经接上。
- 当前应用默认处理当前登录用户的 `.codex` 目录，Windows 下通常是 `%USERPROFILE%\.codex`；工作区也支持显式更换扫描根目录。
- 当前没有多 profile 切换。
- 仓库里现在有两条相关但不同的流程：`重复治理预览` 负责只读扫描和重复计划预览，`真实历史删除` 负责按会话生成计划、审批、备份、删除、回滚和校验。
- 当前交付目标仍然是 Windows 本地工具，不做云端同步，也不做 macOS / Linux 交付。

## 现在能做什么

- 只读扫描当前用户的 `.codex`，生成 `discovery.json`、`manifest-before.json`、`unknown-items.json`。
- 基于只读结果生成重复组和删除计划预览，先看清有哪些疑似重复对象。
- 从本地 `state_5.sqlite` 读取会话列表，支持按标题、目录和 `archived` 状态筛选。
- 对选中的会话生成真实删除计划，覆盖 SQLite、`history.jsonl`、`session_index.jsonl`、全局状态文件、rollout 文件和 shell snapshot。
- 执行前必须先生成 approved plan，并在界面里输入 `purge-selected` 才能继续。
- 支持两种执行模式：`只做备份` 和 `执行真实删除`。
- 删除后会生成 `rollback-journal.json`、`exec-result.json`、`manifest-after.json`；执行失败时会自动回滚，也可以手动按 journal 回滚。
- 可以导出 evidence pack，把本次运行产物和项目架构、领域上下文及迁移历史一起整理成证据清单。

## 现在还没做到的

- 它还不是一个通用清理器，只围绕 Codex 当前这套本地数据结构工作。
- 目前只扫一个根目录，不会自动发现浏览器缓存、WebView 用户目录或别的账号目录。
- 没有独立 CLI 入口，主要使用方式是桌面应用。
- 如果本机 `.codex` 数据结构和当前代码假设不一致，后端会直接报错，不会静默兜底。

## 目录大意

- `frontend/`：桌面界面，React + TypeScript
- `internal/discovery/`：只读扫描和发现产物
- `internal/planning/`：重复分组和预览计划
- `internal/history/`：真实历史列表、审批、执行、回滚、证据导出
- `smoke/`：最小闭环 smoke 测试，使用临时 `.codex` 样本，不碰真实数据

## 本地开发

### 依赖

- Go 1.25（见 `go.mod`）
- Node.js / npm
- Wails CLI 2.x
- Windows 环境下可用的 WebView2 运行时

### 常用命令

```bash
go test ./...
npm --prefix frontend install
npm --prefix frontend run build
wails dev
wails build -clean
```

- `wails dev`：启动桌面开发模式
- `wails build -clean`：生成 Windows 可执行文件 `build/bin/codex-history-manager.exe`

## 运行产物

- 只读扫描产物：`%TEMP%\codex-history-manager\runs\<run-id>\`
- 真实删除产物：`%TEMP%\codex-history-manager\history-runs\<run-id>\`
- 打包产物：`build/bin/codex-history-manager.exe`

## 已验证

2026-07-03 在当前仓库执行通过：

- `go test ./...`
- `npm --prefix frontend run build`
- `wails build -clean`

测试覆盖上，`internal/history` 已覆盖真实删除和回滚，`smoke/` 覆盖只读扫描、计划生成、approved gate、backup-only 执行和 evidence pack 导出。
