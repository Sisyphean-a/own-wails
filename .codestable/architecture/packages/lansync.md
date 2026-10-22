# LanSync

`scope: package:lansync`

## 职责

`LanSync/` 是独立的 Go + Wails + React 局域网文件同步应用，目标平台为 Windows。业务规则由 [局域网同步上下文](../../requirements/contexts/lan-sync.md) 唯一拥有，不复用 RepoMirror 的单向 Git 镜像语义。

## 实现边界

- `main.go`：独立数据目录、Wails 单实例窗口锁、WebView 数据目录和宿主装配。
- `app.go`：Wails 公开动作、目录选择、托盘窗口生命周期及系统授权入口；只在 `OnStartup` 初始化引擎，生成绑定不访问用户数据。前端绑定必须重新生成。
- `internal/syncer/engine.go`、`heartbeat.go`：任务状态、文件监听、定期核对、双任务调度和退出资源回收。
- `network.go`、`identity.go`、`discovery.go`：TLS 设备身份、双端配对、目录邀请、按任务授权、mDNS 和地址更新。发现记录不授予权限，更新已配对地址前仍须校验证书指纹。
- `files.go`、`sync.go`、`merge.go`、`recovery.go`：扫描、Git 排除、哈希、写前比较、备份、临时写入、原子替换、共同版本、冲突保留与中断清理。
- `ai.go`：本机 DeepSeek 配置、单请求生命周期、固定官方 HTTPS 端点、JSON 建议校验和分析前后版本校验；只返回建议，不执行同步副作用。`internal/credentials/` 隔离 Windows DPAPI，非 Windows 明确不支持凭证持久化。
- `store.go`：本地 SQLite WAL 持久化。日常核对只加载共同版本元信息，Markdown 双端修改时才读取共同正文；共同正文与用户可清理的历史备份不是同一生命周期。
- `internal/platform/`：Windows 托盘、当前用户开机启动、防火墙 UAC 和打开目录；非 Windows 的系统集成明确返回不支持。
- `frontend/src/`：目录行快速同步、待处理文档筛选、设备与设置弹层、历史版本；`ConflictWorkspace.tsx` 集中人工差异编辑、DeepSeek 外发同意、策略与正文预览，再交给既有保存入口。`frontend/wailsjs/` 是生成物。

## 协议与数据边界

每个目录对由创建者设备协调，另一端的立即同步与冲突处理请求转交协调端，避免同时规划同一目录对。传输不依赖创建者的固定地址，身份与地址分离。

网络控制请求上限为 8 MiB，文件流独立传输并进行大小与哈希校验。网络仅接受局域网、链路本地或回环来源。文件请求只接受受保护根目录内的普通相对路径，不接收任意绝对路径；拒绝 Git 元数据、保留临时命名空间、链接与不支持的跨平台文件名。

可选 DeepSeek 是单独的出网边界，不改变局域网配对协议；凭证、模型配置与请求仅属于发起设备。DeepSeek 的官方模型与请求格式由 `ai.go` 维护，拒绝重定向和服务端错误正文回显，避免凭证或文档经错误路径扩散。外发与确认规则见[局域网同步上下文](../../requirements/contexts/lan-sync.md)。

设备证书和私钥独立持久化，已有身份不被自动替换；首次并发启动也不能截断覆盖现有密钥。状态默认位于系统配置目录 `LanSync/`，备份根可另选，历史索引保留原备份位置。

Windows 防火墙规则限定可执行文件、专用网络和本地子网；TCP 按应用限定以适应动态端口，设备发现只开放 UDP 5353。不使用管理员身份常驻，不放开其他程序或配置路由器映射。

## 验证与剩余边界

聚焦证据为 `internal/syncer/*_test.go`、`frontend/src/diff.test.ts`、`go test -race ./...`、`go vet ./...`、前端构建和 `wails build -clean`。界面验证使用真实 Wails 后端，不以模拟数据证明文件同步。

DeepSeek 测试使用受控 HTTP 边界，不调用付费服务；新版界面已用真实 TLS 双端验证冲突保留、确认保存、密钥增删与窄窗口布局，AI 建议展示使用受控响应。真实云端模型与账户联调仍待有效密钥验证。

符号链接动态测试因 Windows 创建链接权限不足而跳过，仍待验证。真实双机 mDNS、系统授权、开机启动与休眠唤醒尚待实机联调。高风险最终差异仍需用户另开独立会话或人工审查，不以实现会话的自查替代。
