# 项目记忆入口

`.codestable/` 是整个单仓库的唯一默认记忆入口。开始工作时先读取本文件、`architecture/INDEX.md`，再按目标子项目读取对应架构页和领域上下文。

## 当前硬边界

- 四个子项目是四个独立的 Go/Wails 应用；不要把它们当成一个共享 Wails 入口或一个统一前端工程。
- 子项目各自的代码、测试和当前配置优先于旧记忆；根记忆只保存会影响后续判断的稳定边界、术语和原因。
- Go/Wails 绑定是各应用跨端契约的源头；`frontend/wailsjs/` 是生成物，修改绑定后必须重新生成并验证构建。
- 破坏性操作必须保留真实错误和可追溯结果：Codex 历史删除要有批准、备份、回滚和执行后校验；GistSync 恢复要有加密、冲突和显式覆盖边界；RepoMirror 不得触碰 Git 元数据和受保护路径；logcat 的 ADB、解析、过滤和导出错误不得伪造成功。

## 最小验证

- `codex-history-clear/`：`go test ./...`、`npm --prefix frontend run build`、`wails build -clean`
- `RepoMirror/`：`go test ./...`、`npm --prefix frontend run build`、`wails build -clean`
- `GistSync/`：`go test ./...`、`npm --prefix frontend run build`、`wails build -clean`
- `logcat/`：`go test ./...`、`npm --prefix frontend run build`、`wails build -clean`

记忆页面只记录已验证的术语、稳定规则、架构边界和重要原因；任务过程、截图、生成产物和可由代码直接表达的普通实现细节不进入这里。
