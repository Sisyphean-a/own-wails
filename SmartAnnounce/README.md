# SmartAnnounce

基于 `Wails + Go + Vue 3` 的桌面端超市播报生成器。输入促销文案、选择音色后，应用会调用 Minimax TTS 接口生成语音，保存到本地 `Downloads/SmartAnnounce` 目录，并在应用内直接试听、循环播放和另存为导出。

## 环境要求

- Go 1.23+
- Node.js 20+
- Wails CLI 2.11+
- Windows WebView2 运行时

## Minimax 配置

首次打开应用时，在弹出的“Minimax 设置”中输入 API Key，必要时修改 TTS 模型（默认为 `speech-2.8-hd`），点击“保存配置”。之后可通过窗口右上角“设置”修改；已经保存 Key 时留空可只修改模型，输入新 Key 则替换旧 Key。

Minimax TTS 请求默认**直连**，不会跟随系统或环境变量中的代理。如果需要代理，在设置中选择“自定义代理”并填写 `http://127.0.0.1:7890` 这类带端口的 HTTP/HTTPS 地址；不支持含用户名密码的代理 URL。切回“直连”即可停用自定义代理，代理不可用时会显示真实请求错误，不会偷偷回退到直连。已有配置升级后保持原 Key/模型，代理默认为直连。

配置保存在当前用户的应用配置目录 `SmartAnnounce/minimax.json`。API Key 使用 Windows 当前用户的 DPAPI 加密，不会以明文写入文件或回传到配置界面；切换 Windows 用户或重装系统后可能无法解密，需要重新配置。应用不再读取 `MINIMAX_API_KEY` 或 `MINIMAX_TTS_MODEL` 环境变量，已有环境变量不会自动迁移。配置文件损坏或解密失败时会明确报错，不会忽略错误继续生成。

## 开发

```powershell
npm install --prefix frontend
wails dev
```

## 构建

```powershell
wails build -nopackage
```

构建产物位于 `build/bin/SmartAnnounce.exe`。

## 当前实现

- 深色桌面端双栏界面与应用内 Minimax 设置弹窗
- 8 个普通话音色（默认成熟女性音色，含新闻女声）、生成语速与生成音量调节
- Go 端调用 Minimax `v1/t2a_v2` 接口生成 MP3
- 生成结果按“文案片段 + 高精度时间戳”落盘保存并回传 Base64 Data URI
- 应用内播放、暂停、进度拖拽、循环播放、播放音量控制
- 原生保存对话框导出音频到任意位置，若目标文件已存在则显式报错
