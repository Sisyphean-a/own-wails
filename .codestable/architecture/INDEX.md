# 架构索引

## 范围地图

| 范围 | 当前入口 | 负责内容 |
| --- | --- | --- |
| `workspace` | `README.md`、本页 | 三个独立 Wails 应用的单仓库组织、构建入口和记忆边界 |
| `package:codex-history-clear` | [codex-history-clear.md](packages/codex-history-clear.md) | Codex 本地历史发现、重复分析、删除计划、备份、回滚和证据导出 |
| `package:repomirror` | [repomirror.md](packages/repomirror.md) | 两个 Git 仓库之间的差异计算、镜像、提交和推送 |
| `package:gist-sync` | [gist-sync.md](packages/gist-sync.md) | 配置集、加密快照、GitHub Gist 同步、冲突预检和恢复 |

## 运行拓扑

```text
own-wails/
  ├─ codex-history-clear/  -> 独立 Wails + Go + React 应用
  ├─ RepoMirror/           -> 独立 Wails + Go + React 应用
  └─ GistSync/             -> 独立 Wails + Go + Vue 应用
```

根目录不提供共享 Wails 宿主。每个子项目分别嵌入自己的 `frontend/dist`，分别解析自己的 `wails.json`，并使用自己的 Go 模块和前端锁文件。

## 公共边界

- 子项目之间目前没有确认过的业务共享协议；相同的 `main.go`、`app.go`、`frontend/` 等路径只是各自项目内部的同名入口，不应因此抽取共享代码。
- 根级 `.codestable/` 只负责范围地图和长期规则；具体实现事实仍以对应子项目代码、测试和配置为准。
- 需要跨应用复用机制时，先确认语义和生命周期确实相同，再新增明确的共享包或共享契约。

## 按范围加载

- 修改仓库组织、根脚本或记忆入口：读取 `attention.md`、本页和 `requirements/CONTEXT.md`。
- 修改 `codex-history-clear/`：额外读取 [codex-history-clear.md](packages/codex-history-clear.md) 和 [history-governance.md](../requirements/contexts/history-governance.md)。
- 修改 `RepoMirror/`：额外读取 [repomirror.md](packages/repomirror.md) 和 [repomirror.md](../requirements/contexts/repomirror.md)。
- 修改 `GistSync/`：额外读取 [gist-sync.md](packages/gist-sync.md) 和 [gist-sync.md](../requirements/contexts/gist-sync.md)。
