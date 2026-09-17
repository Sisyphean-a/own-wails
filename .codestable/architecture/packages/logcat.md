# logcat

`scope: package:logcat`

## 职责

`logcat/` 是一个 Windows Go + Wails + React 桌面应用，用于通过本机 ADB 实时查看 Android Logcat，支持设备、包/进程绑定、过滤、搜索、保存过滤器、详情解析、复制和 TSV 导出，重点服务 H5/WebView 调试场景。

运行桌面功能依赖本机 `adb` 和已连接的 Android 设备；浏览器预览使用模拟数据，不能证明 ADB、导出或桌面剪贴板行为。

## 代码边界

- `logcat/main.go`：装配 `adb.Service`、`adb.LogcatSource`、`session.Supervisor`、`app.Controller`，恢复保存的过滤器并绑定根 `App`。
- `logcat/app.go`：唯一 Wails 公开边界，处理动作、剪贴板、导出、过滤器持久化和状态投影。
- `logcat/internal/app/`：设备选择、包/PID 绑定、过滤、搜索、暂停、选择和有界日志缓存。
- `logcat/internal/adb/`：ADB 命令、设备跟踪、包/进程发现和日志流。
- `logcat/internal/session/`：把日志源转成解析事件并管理会话生命周期。
- `logcat/internal/logcat/`：解析 `threadtime` 行和 Chromium/WebView source。
- `logcat/internal/storage/`：过滤器持久化和 TSV 导出。
- `logcat/frontend/src/`：React 界面和状态流消费；`frontend/wailsjs/` 是生成代码。

## 稳定规则

- 同一时间只能有一个活跃日志会话；设备、包、进程或 PID 绑定变化时，旧会话与 watcher 必须先停止，避免日志混流。
- `Controller` 以 `SourceIndex` 表示日志身份；过滤、搜索、选择和增量同步不能依赖当前列表下标。
- 设备跟踪合并重复序列号，并且只保留或自动选择状态为 `device` 的设备。
- 包相关进程仅指进程名等于包名，或以 `包名:` 开头的进程。
- 清空只清空本地日志视图和选择，不执行 `adb logcat -c`，不修改设备日志缓冲。
- 后端日志缓存上限为 100000 条；前端传输窗口最多 1000 条，这是两个不同边界。
- ADB、解析、过滤和导出错误必须明确呈现或返回，不能伪造成功或静默继续。
- 导出只处理当前可见日志为 TSV；没有可见日志时必须报错。
- 日志语义 token 化只改变呈现，必须保留原文本的空白和位置偏移。

## 查询与本地数据

过滤器字段为 `level`、`tag`、`message`、`package`；支持 `:`/`=`、`~:`/`~=`、前缀 `-`、`&`/`&&`、`|`/`||`、括号和双引号值。无字段裸词按消息文本执行不区分大小写的包含匹配。

保存的过滤器、默认过滤器和查询历史写入系统配置目录下的 `logcat-viewer/saved-filters.json`；导出日志写入用户 `Downloads` 目录。

## Wails 状态流协议

前端先通过 `GetState()` 获取完整 `AppState`，随后订阅 `state:updated` 和 `state:append`。完整状态是可恢复基线；后端只有在上下文和窗口重叠均验证一致时才能发送 `StateAppendPatch`，否则必须回退为完整状态。仅影响选择的动作返回 `SelectionPatch`。

三类载荷都通过 `Revision` 和 `SourceIndex` 协作。前端只能在已有完整基线上应用补丁；缺少基线、上下文不一致或载荷无法验证时必须等待/请求全量状态，不能猜测合并。

该协议的决定来源为 ADR-001；实现锚点是 `logcat/app.go`、`app_state*.go`、`selection_patch.go`、`frontend/src/use-app-controller.ts` 和 `frontend/src/state-stream.ts`。

## 验证入口

- `logcat/internal/adb/*_test.go`
- `logcat/internal/app/*_test.go`
- `logcat/internal/logcat/*_test.go`
- `logcat/internal/session/*_test.go`
- `logcat/internal/storage/*_test.go`
- `go test ./...`
- `npm --prefix frontend run build`
- `wails build -clean`
