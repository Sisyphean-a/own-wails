# Codex 本地历史治理

`scope: context:history-governance`

## 范围

本上下文描述 `codex-history-clear/` 对 Windows 当前用户本地 `.codex` 数据的发现、重复识别、计划和可回滚治理。它不覆盖云端 Conversations/Responses、浏览器缓存、WebView 用户目录或其他账号的同步治理。

## 规范语言

| 术语 | 定义 | 避免混用 |
| --- | --- | --- |
| **CODEX_HOME** | 当前用户 Codex 本地状态根目录；默认是用户目录下的 `.codex`，也可由工作区绑定显式设置 | 安装目录、程序目录 |
| **全局历史** | 跨会话追加式消息历史，典型对象是 `history.jsonl` | 会话转录、会话索引 |
| **会话转录** | 单会话消息序列，典型对象是 `rollout-*.jsonl` | 全局历史、诊断日志 |
| **活动数据库** | 当前客户端可能占用的 SQLite 状态数据库对象 | 备份副本、导出快照 |
| **候选根目录** | 在真实存储未确认时供发现阶段探测的一组本地根路径 | 真源目录、权威目录 |
| **删除计划** | destructive 动作前的结构化目标、存储和动作清单 | 运行脚本、临时命令 |
| **隔离区** | 待永久清除对象在执行链中的本地暂存位置 | 回收站、最终归档 |
| **回滚日志** | 记录 destructive 前后路径映射与执行结果的审计文件 | 运行日志、调试输出 |
| **保留本** | 重复组中被选为权威保留对象的记录或文件 | 任意副本、最新文件 |
| **路径别名** | 指向同一工程或物理对象的不同路径表示，如 Win32、WSL、junction 或 symlink 入口 | 物理重复、真复制 |
| **逻辑重复** | 指向同一会话内容但来自不同索引、展示入口或路径表示的重复记录 | 物理重复 |
| **物理重复** | 真实复制出的多份文件对象 | 路径别名、索引漂移 |
| **桌面壳层** | 基于 Wails2 的本地桌面宿主，承载前端、窗口生命周期和后端绑定 | 浏览器页面、云端控制台 |
| **执行任务** | 以 `run_id` 标识的一次扫描、计划或执行作业及其进度与证据集合 | 单个命令、临时操作 |

## 稳定规则与不变量

1. 发现阶段只读扫描当前解析出的工作区根，并将对象、manifest 和未知项落成可复核产物；未知对象不因无法分类而静默删除。
2. 重复识别区分路径别名、逻辑重复和物理重复；同一会话的多入口不自动等于多份可删除文件。
3. 预览计划不是执行授权。真实删除必须加载 approved plan、重新校验当前目标意图、确认目标未活动，并由 Go 服务执行。
4. 默认执行先备份；`backup-only` 只能生成备份并跳过 destructive 改写。真实执行必须留下 rollback journal、exec result 和执行后 manifest/verification。
5. 真实执行失败时优先保留真实错误并按 rollback journal 恢复；不能用空结果、静默跳过或伪造成功掩盖不一致。
6. 前端只负责展示、筛选、选择和确认，不能直接访问 SQLite/JSONL、拼接删除命令或绕过后端校验。

## 当前实现锚点

- 发现：`codex-history-clear/internal/discovery/service.go#Service.RunReadOnlyScan`、`roots.go#classifyPath`。
- 重复预览：`codex-history-clear/internal/planning/service.go#Service.BuildDeletePlan`、`grouping.go#groupKeyFor`。
- 真实治理：`codex-history-clear/internal/history/service.go#Service.ExecuteDeletePlan`、`execute.go#executeDeletePlan`、`rollback.go#rollbackExecution`。
- 桌面边界：`codex-history-clear/workspace_binding.go`、`history_binding.go`、`frontend/src/history-workspace-controller.ts`。
