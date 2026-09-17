# codex-history-clear

`scope: package:codex-history-clear`

## 职责

`codex-history-clear/` 是 Windows 本地 Codex 历史治理桌面应用。它解析 `CODEX_HOME`，生成只读发现和重复预览，构造经过批准的删除计划，并执行带备份、回滚和执行后校验的历史清理。

技术边界是 Go + Wails 2 + React/TypeScript；当前不提供云端同步、跨平台交付、独立 CLI 或多 profile 管理。

## 代码边界

- `codex-history-clear/main.go#main`、`app.go#App`：Wails 生命周期、服务装配和前端绑定宿主。
- `codex-history-clear/internal/codexhome#Resolve`：默认用户目录与显式工作区根的解析、存在性和目录校验。
- `codex-history-clear/internal/discovery`：只读扫描 `.codex`，识别 `config.toml`、认证文件、历史 JSONL、SQLite、session index 和 rollout，写出发现产物、manifest 和未知项。
- `codex-history-clear/internal/planning`：读取 manifest，按会话/线程、工作目录、路径和内容证据分组，输出重复组和删除计划预览。
- `codex-history-clear/internal/history`：列表、计划、批准、备份、执行、回滚、复扫、验证和证据导出。
- `codex-history-clear/scan_binding.go`、`plan_binding.go`、`history_binding.go`、`workspace_binding.go`：只做 DTO 映射和服务调用，不承载存储逻辑。
- `codex-history-clear/frontend/`：React 工作台；只展示状态、收集筛选/选择/确认并调用 Wails 绑定。

## 数据流与安全边界

1. `RunReadOnlyScan` 解析一个工作区根并只读收集对象、manifest 和未知项。
2. 列表和计划阶段重新校验活动数据库、关联存储和目标转录，避免把旧预览当成真源。
3. `ApproveHistoryDeletePlan` 将计划转换为 approved plan；执行前再次比较计划意图和活动目标状态。
4. `ExecuteHistoryDeletePlan` 要求 approved plan、用户确认和非活动目标；默认先备份，`BackupOnly` 只生成备份而不修改目标。
5. destructive 执行写 rollback journal、exec result、执行后 manifest 和 verification；失败时按 journal 恢复并保留真实错误。
6. 前端不能直接访问 SQLite/JSONL、拼接删除命令或绕过 Go 校验。

领域术语和不变量见 [history-governance.md](../../requirements/contexts/history-governance.md)。

## Wails 跨端契约

Go 绑定和 DTO 是契约源头：`codex-history-clear/scan_binding.go`、`plan_binding.go`、`history_binding.go`、`workspace_binding.go` 及 `internal/*/types.go`。`frontend/wailsjs/go/main/App.*` 和 `frontend/wailsjs/go/models.ts` 是生成的前端适配物，不是第二份业务规则定义。

绑定返回稳定 JSON DTO；前端不接收可执行命令，也不获得绕过 Go 校验的系统访问权限。新增字段或重命名时必须同步更新 Go 类型、绑定、前端调用和测试，并运行后端、前端及 Wails 构建验证。

## 前端交互流

```text
加载工作区 -> 只读扫描/预览 -> 列出会话与重复建议
  -> 筛选并选择 -> 生成计划 -> 批准 -> 输入确认短语
  -> 备份-only 或真实执行 -> 展示 journal / verification / evidence
```

## 验证入口

- `codex-history-clear/internal/discovery/*_test.go`
- `codex-history-clear/internal/planning/service_test.go`
- `codex-history-clear/internal/history/*_test.go`
- `codex-history-clear/smoke/history_flow_test.go`
- `go test ./...`
- `npm --prefix frontend run build`
- `wails build -clean`
