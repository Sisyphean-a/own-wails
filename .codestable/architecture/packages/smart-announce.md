# SmartAnnounce

`scope: package:smart-announce`

## 职责

`SmartAnnounce/` 是 Go + Wails + Vue 3 的 Windows 桌面超市播报生成器。用户输入促销文案、选择中文音色和语速/音量后，应用调用 Minimax TTS 生成 MP3，保存到用户 `Downloads/SmartAnnounce`，并提供应用内试听、暂停、进度拖拽、循环播放和另存为导出。

## 代码边界

- `SmartAnnounce/main.go`：Wails 入口、窗口布局和资源嵌入。
- `SmartAnnounce/app.go`：唯一 Wails 绑定边界，负责生成语音、打开保存对话框和导出。
- `SmartAnnounce/internal/broadcast/`：请求校验、Minimax TTS 客户端、音频存储、MP3 元数据和导出。
- `SmartAnnounce/frontend/src/App.vue`：页面状态、生成/导出动作和 HTML Audio 播放控制。
- `SmartAnnounce/frontend/src/components/`：播报编辑器、音色卡片和试听播放器。
- `SmartAnnounce/frontend/wailsjs/`：Wails 生成的调用与模型代码，不是手写契约。

## 外部边界与安全规则

- 运行生成语音需要环境变量 `MINIMAX_API_KEY`；可选 `MINIMAX_TTS_MODEL`，默认模型为 `speech-2.8-hd`。
- API Key 只从进程环境读取，不写入本地文件、前端状态或导出结果。
- 未配置 API Key、网络失败、HTTP 非成功状态、业务失败、响应解析失败或音频解码失败都必须返回真实错误，不得静默降级。
- TTS 请求固定使用 MP3、单声道、32000Hz、128kbps，并把返回的十六进制音频解码后保存。
- 播报文本去除首尾空白，最多 2000 个 Unicode 字符；必须选择音色，语速范围为 `0.5` 到 `2.0`，生成音量范围为 `0` 到 `1`。
- 生成音频保存到本地 `Downloads/SmartAnnounce`；导出目标由原生保存对话框选择，空路径表示取消，已有目标文件必须显式报错。
- Base64 Data URI 只用于当前界面试听，不替代本地文件路径作为导出来源。

## 前端行为

页面分为文案编辑和试听播放器两栏。生成成功后重置播放位置并加载新音频；播放、暂停、循环、播放音量和进度由 HTML Audio 驱动。生成或导出失败显示后端错误，未生成音频时不能播放或导出。

## 验证入口

- `SmartAnnounce/internal/broadcast/*_test.go`
- `SmartAnnounce/window_config_test.go`
- `go test ./...`
- `npm --prefix frontend run build`
- `wails build -clean`

真实 Minimax 请求和真实音频播放依赖外部 API Key、网络和 Windows 音频环境；自动化验证不应把这些外部条件伪装成成功。
