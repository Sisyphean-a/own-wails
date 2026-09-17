# GistSync

`scope: package:gist-sync`

## 职责

`GistSync/` 是 Go + Wails + Vue 的 Windows 桌面应用，用配置集选择本地文件，以主密码加密后写入 GitHub Gist，并按快照恢复到另一台电脑。

## 代码边界

- `GistSync/main.go`、`app.go`：Wails 启动、生命周期和前端绑定。
- `GistSync/internal/appsvc/`：编排设置、配置集管理和同步用例，通过接口隔离 `settings` 与 `syncflow`。
- `GistSync/internal/settings/`：配置集元数据持久化、旧格式迁移和系统凭证存取；是 Token、主密码和配置状态的唯一持久化入口。
- `GistSync/internal/syncflow/`：Gist 清单、快照、加密上传、冲突预检和按条目恢复。
- `GistSync/internal/gistapi/`：GitHub Gist API 适配。
- `GistSync/internal/security/`：加解密。
- `GistSync/internal/pathmap/`、`profileutil/`：路径模板、标识和相对路径规范化。
- `GistSync/frontend/`：Vue 页面、状态协调和 Wails 调用；不直接访问文件系统、凭证库或 GitHub API。

## 领域模型

- **配置集（Profile）**：一组要同步的本地配置文件及其恢复策略；应用始终至多有一个活动配置集。
- **条目（ProfileItem）**：配置集内带稳定 ID、源路径模板、相对路径和启用状态的一项本地文件。
- **快照（Snapshot）**：一次上传产生的、属于一个配置集的加密文件引用集合。
- **清单（Manifest）**：Gist 中的 `sync_manifest.json`，保存配置集和快照元数据。
- **原路径恢复（original）**：按条目源路径恢复。
- **指定根目录恢复（rooted）**：按条目相对路径写入用户提供的根目录。

## 同步与恢复规则

1. 同步内容以“配置集 + 条目 + 快照”组织，云端不保存文件明文。
2. 只有启用且被选择的条目可进入上传；恢复也可按条目选择。
3. 每次上传为配置集新增快照；快照条目指向独立加密 blob。
4. 每次从云端读取配置集或快照都必须读取当前 Gist 清单，不能以本机缓存替代；启动与刷新先把云端配置集定义合并到本地。
5. 上传由 `appsvc` 解析本地设置、Token 和主密码，再调用 `syncflow` 读取文件、以 AES-256-GCM 加密并写入 Gist。
6. 没有主密码不能上传或恢复；主密码错误或解密失败时不得写入目标文件。
7. 既有目标文件默认不覆盖；只有调用者显式授权该条目时才能覆盖。冲突预检负责展示目标存在和差异。
8. `rooted` 恢复必须有非空根目录；否则请求无效。
9. 单个恢复条目失败只记录为该条目的 `error` 结果，其他已选条目继续处理；成功写入会创建父目录并使用 `0600` 权限。

## 凭证与本地设置

- 默认设置路径是 `os.UserConfigDir()/GistSync/settings.json`。
- GitHub Token 和主密码由 `go-keyring` 系统凭证库保存；设置 JSON 只保存凭证引用，不写入明文凭证。
- 设置文件使用 `0600` 权限写入；manifest 本地缓存位于 `os.UserCacheDir()/GistSync/manifest-cache`，有效期为 30 秒。
- 读取旧格式设置时由 `internal/settings` 负责凭证字段和旧 `syncPath` 的迁移，上层只使用 `settings.Data`。
- 编辑 Token 或主密码时，候选主密码必须经过只读云端快照验证；无法验证、凭证/网络失败或验证未完成时不得写入系统凭证库。没有云端快照时可以保存，但必须明确提示尚不可验证。

## 前端流程

Vue 前端提供一键同步、高级同步、配置管理和安全设置四个界面。`useSettingsStore` 负责共享设置状态和失败回滚，`useQuickSyncStore` 负责一键上传/下载及冲突确认，`SyncCenter.vue` 负责条目选择、快照选择、冲突预检和恢复，`src/lib/backend.ts` 是前端模型与 Wails 生成模型之间的唯一映射层。

## 验证入口

- `GistSync/internal/appsvc/*_test.go`
- `GistSync/internal/settings/*_test.go`
- `GistSync/internal/syncflow/*_test.go`
- `GistSync/frontend/tests/*`
- `go test ./...`
- `npm --prefix frontend run build`
- `wails build -clean`
