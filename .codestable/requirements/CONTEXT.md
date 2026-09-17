# 需求与领域范围地图

本文件只做单仓库作用域入口，不重复定义各领域术语。默认先读取它，再进入目标子项目的直接上下文。

## 工作区范围

`workspace` 覆盖根目录的四个独立 Wails 子项目、仓库级组织和构建入口。整体架构与包归属见 [architecture/INDEX.md](../architecture/INDEX.md)。

## 领域上下文

| 上下文 | 所有者 | 适用范围 | 入口 |
| --- | --- | --- | --- |
| `history-governance` | Codex 本地历史治理能力 | `codex-history-clear/` 的发现、重复识别、计划、批准、备份、删除、回滚和证据 | [history-governance.md](contexts/history-governance.md) |
| `repomirror` | RepoMirror 镜像同步能力 | `RepoMirror/` 的仓库选择、差异、保护路径、同步、提交和推送 | [repomirror.md](contexts/repomirror.md) |
| `gist-sync` | GistSync 配置同步能力 | `GistSync/` 的配置集、加密快照、凭证、冲突和恢复 | [gist-sync.md](contexts/gist-sync.md) |
| `logcat` | Android 日志查看能力 | `logcat/` 的 ADB 采集、会话、过滤、状态流和导出 | [logcat.md](contexts/logcat.md) |

包是实现边界；领域上下文只由语义所有权决定。四个子项目之间没有默认共享的业务术语或跨包协议。

## 作用域规则

- 一个术语或不变量只在最窄的拥有页面定义；其他页面链接，不复制定义。
- 当前代码和测试是实现事实的校验依据；`history/` 只承载变化原因、迁移来源和仍需追溯的风险。
- 记忆目录只有根级 `.codestable/`；子项目不得重新引入各自的 `.codestable/`。
