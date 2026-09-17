# SmartAnnounce

基于 `Wails + Go + Vue 3` 的桌面端超市播报生成器。输入促销文案、选择音色后，应用会调用 Minimax TTS 接口生成语音，保存到本地 `Downloads/SmartAnnounce` 目录，并在应用内直接试听、循环播放和另存为导出。

## 环境要求

- Go 1.23+
- Node.js 20+
- Wails CLI 2.11+
- Windows WebView2 运行时

## 必要环境变量

运行前必须设置：

```powershell
$env:MINIMAX_API_KEY="你的 Minimax API Key"
```

可选：

```powershell
$env:MINIMAX_TTS_MODEL="speech-2.8-hd"
```

未设置 `MINIMAX_API_KEY` 时，点击“生成语音”会直接报错，不做静默降级。

## 开发

```powershell
npm install --prefix frontend
wails dev
```

## 构建

```powershell
wails build -nopackage
```

构建产物位于 [build/bin/SmartAnnounce.exe](F:\Github\SmartAnnounce\build\bin\SmartAnnounce.exe)。

## 当前实现

- 深色桌面端双栏界面，布局参考设计稿
- 4 个中文音色预设、生成语速与生成音量调节
- Go 端调用 Minimax `v1/t2a_v2` 接口生成 MP3
- 生成结果按“文案片段 + 高精度时间戳”落盘保存并回传 Base64 Data URI
- 应用内播放、暂停、进度拖拽、循环播放、播放音量控制
- 原生保存对话框导出音频到任意位置，若目标文件已存在则显式报错
