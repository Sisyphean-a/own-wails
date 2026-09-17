# Smoke Fixtures

本目录记录 `go test ./smoke/...` 使用的最小闭环夹具约定。

- 测试会在临时目录动态生成 `.codex` 样本，而不是直接改仓库内文件。
- 成功路径覆盖：只读扫描、重复计划、历史删除计划、approved gate、backup-only 执行、evidence pack 导出。
- 错误路径覆盖：未批准计划直接执行会被后端显式阻断。
