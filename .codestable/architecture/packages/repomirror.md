# RepoMirror

`scope: package:repomirror`

## 职责

`RepoMirror/` 是一个 Go + Wails + React 的 Windows 桌面工具，用于在两个本地 Git 仓库目录之间执行单向镜像同步。它不是 Git 客户端，也不是 merge/rebase 工具；双向能力只表示可以切换 `A → B` 或 `B → A`，每次执行仍只有一个明确方向。

## 核心模块

- `RepoMirror/main.go`：组装依赖并创建 Wails 应用。
- `RepoMirror/app.go`、`RepoMirror/internal/app/`：Wails 绑定层与应用服务编排。
- `RepoMirror/internal/config/`：配置文件读写，默认路径为 `%APPDATA%/RepoMirror/config.json`。
- `RepoMirror/internal/gitops/`：通过本机 `git` 命令执行仓库根解析、文件列表、ignore、状态、分支、commit 和 push。
- `RepoMirror/internal/diff/`：基于源端文件集和目标端保护规则计算差异。
- `RepoMirror/internal/syncer/`：按差异结果复制或删除目标仓库文件。
- `RepoMirror/internal/platform/`：文件系统和文本比较抽象。
- `RepoMirror/frontend/src/`：控制区、差异列表和目标状态区组成的 React 界面。

## 核心术语

- `Direction`：`A_TO_B` 或 `B_TO_A`。
- `SourceRoot`：当前方向下的源仓库根目录。
- `TargetRoot`：当前方向下的目标仓库根目录。
- `DiffEntry`：`added`、`modified` 或 `deleted` 的单个差异文件。
- `TargetRepositoryStatus`：目标仓库分支、干净状态、未提交数量和未跟踪数量。

## 稳定规则与硬边界

- Git 交互统一通过本机 `git` 命令完成，不引入额外 Git SDK。
- 差异计算不依赖时间戳；文件同时存在时按实际内容比较。
- 源仓库输出集合采用 `git ls-files --cached --others --exclude-standard -z`。
- 目标仓库保护集合包括 `.git`、任意层级 `.gitignore` 和 `git check-ignore` 命中的路径。
- `.git` 目录内容绝不复制、删除或覆盖；`.gitignore` 文件不进入差异列表，也不参与同步。
- 常见文本文件的纯 `LF`/`CRLF` 差异不算修改；二进制文件仍按字节比较。
- 目标仓库已有未提交改动时仍可同步，但必须显式展示状态，由用户决定。
- commit 和 push 只作用于当前目标仓库；空提交信息、非法路径、非 Git 仓库和 Git 命令错误必须显式失败。
- 当前不实现 merge、rebase、stash、冲突解决、分支管理或 commit 历史浏览。

## 验证入口

- `RepoMirror/internal/config/*_test.go`
- `RepoMirror/internal/diff/*_test.go`
- `RepoMirror/internal/gitops/*_test.go`
- `RepoMirror/internal/platform/*_test.go`
- `RepoMirror/internal/syncer/*_test.go`
- `go test ./...`
- `npm --prefix frontend run build`
- `wails build -clean`
