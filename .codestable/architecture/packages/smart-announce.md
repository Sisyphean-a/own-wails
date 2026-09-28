# SmartAnnounce

`scope: package:smart-announce`

## 职责

`SmartAnnounce/` 是 Go + Wails + Vue 3 的 Windows 桌面超市播报生成器。用户输入促销文案、选择中文音色和语速/音量后，应用调用 Minimax TTS 生成 MP3，保存到用户 `Downloads/SmartAnnounce`，并提供应用内试听、暂停、进度拖拽、循环播放和另存为导出。

## 代码边界

- `SmartAnnounce/main.go`、`window_config.go`：Wails 入口、默认窗口布局和资源嵌入。
- `SmartAnnounce/app.go`：唯一 Wails 绑定边界，负责配置读取/保存、生成语音、打开保存对话框和导出。
- `SmartAnnounce/internal/broadcast/`：请求校验、Minimax TTS 客户端及仅作用于该客户端的代理策略、当前用户加密配置存储、音频存储、MP3 元数据和导出。
- `SmartAnnounce/frontend/src/App.vue`：文案/音色/参数工作台、生成与导出状态、上一版本标记及 HTML Audio 播放控制；`frontend/src/style.css` 负责桌面分栏、固定操作区和窄屏重排。
- `SmartAnnounce/frontend/wailsjs/`：Wails 生成的调用与模型代码，不是手写契约。

## 外部边界与安全规则

- Minimax 凭证与模型通过应用内设置管理，不再读取 `MINIMAX_API_KEY` / `MINIMAX_TTS_MODEL` 环境变量；模型默认 `speech-2.8-hd`。
- `internal/broadcast/settings.go` 只在 Go 端加载 API Key；`secret_windows.go` 用 Windows 当前用户 DPAPI 加密保存到用户配置目录。配置读取只返回模型与是否配置，不向前端回传已存 Key；空输入保持原 Key，显式输入替换。
- `settings.go` 持久化直连/自定义代理模式和无凭证代理 URL；旧配置缺少代理字段时视为直连。`minimax.go` 对每次 TTS 请求使用配置快照创建独立 HTTP Transport，默认 `Proxy=nil` 不读取系统/环境代理；自定义模式只用显式地址。语义约束见 [SmartAnnounce TTS 播报](../../requirements/contexts/smart-announce.md)。
- 配置文件损坏、无法解密、未配置 API Key、网络失败、HTTP 非成功状态、业务失败、响应解析失败或音频解码失败都必须返回真实错误，不得静默降级或退回环境变量。
- TTS 请求固定使用 MP3、单声道、32000Hz、128kbps，并把返回的十六进制音频解码后保存。
- 播报文本去除首尾空白，最多 2000 个 Unicode 字符；必须选择音色，语速范围为 `0.5` 到 `2.0`，生成音量范围为 `0` 到 `1`。
- 生成音频保存到本地 `Downloads/SmartAnnounce`；导出目标由原生保存对话框选择，空路径表示取消，已有目标文件必须显式报错。
- Base64 Data URI 只用于当前界面试听，不替代本地文件路径作为导出来源。

## 前端行为

页面以文案编辑区、音色/参数设置区和固定试听区组织；右上角弹窗配置 Minimax Key、模型和代理模式，首次缺 Key 时主动打开。默认窗口尺寸与两列音色列表匹配，使初始桌面视口能显示完整工作台。生成按钮保持在设置区底部，设置内容可独立滚动。生成成功后重置播放位置并加载新音频；播放、暂停、循环、播放音量和进度由 HTML Audio 驱动。生成结果与编辑草稿的版本关系由 `frontend/src/App.vue` 对照提交时输入来呈现，语义规则见 [SmartAnnounce TTS 播报](../../requirements/contexts/smart-announce.md)。生成或导出失败显示后端错误，未生成音频时不能播放或导出。

## 验证入口

- `SmartAnnounce/internal/broadcast/*_test.go`
- `SmartAnnounce/window_config_test.go`
- `go test ./...`
- `npm --prefix frontend run build`
- `wails build -clean`

真实 Minimax 请求和真实音频播放依赖外部 API Key、网络和 Windows 音频环境；自动化验证不应把这些外部条件伪装成成功。
