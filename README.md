# own-wails

这是一个包含五个独立 Wails 桌面应用的单仓库。每个子目录都是一个完整项目，拥有自己的 Go 模块、Wails 配置、前端工程、测试和构建产物。

## 子项目

| 目录 | 技术栈 | 用途 |
| --- | --- | --- |
| `codex-history-clear/` | Go + Wails + React | 扫描、计划、备份、删除和回滚 Codex 本地历史 |
| `RepoMirror/` | Go + Wails + React | 在两个 Git 仓库之间执行单向镜像同步 |
| `GistSync/` | Go + Wails + Vue | 加密同步本地配置文件到 GitHub Gist |
| `logcat/` | Go + Wails + React | Android Logcat 实时查看、过滤、详情和导出 |
| `SmartAnnounce/` | Go + Wails + Vue | 基于 Minimax TTS 的超市播报生成和音频导出 |

三个应用保持独立构建，不共享根级前端依赖，也不把三个 Wails 入口合并成一个程序。

## 开发命令

命令需要在对应子项目目录执行：

```powershell
cd codex-history-clear
npm --prefix frontend ci
npm --prefix frontend run build
go test ./...
wails build -clean
```

`RepoMirror`、`GistSync`、`logcat` 和 `SmartAnnounce` 使用相同的 Go、前端和 Wails 检查流程。`GistSync` 的前端构建会额外执行 `vue-tsc`；`logcat` 运行桌面功能还需要本机 `adb`；`SmartAnnounce` 运行生成语音功能需要 `MINIMAX_API_KEY`。

## 仓库边界

- 每个子项目保留自己的 `go.mod`、`go.sum`、`wails.json` 和 `frontend/`。
- 根目录不提供第二套 Go 或 Node 依赖定义。
- `.codestable/` 是整个单仓库唯一的项目记忆入口，按子项目范围组织架构和领域规则。
- `build/bin`、`frontend/dist`、`node_modules`、`.codegraph` 等本地产物不纳入版本控制。
