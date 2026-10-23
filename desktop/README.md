# Eino Multi Agent 桌面端

基于 [Tauri 2](https://v2.tauri.app/) 的壳应用：复用仓库内 `web/` Vue 控制台，连接已启动的 API（默认 `http://127.0.0.1:8180`），不内嵌后端与数据库。

## 前置依赖

- Node.js 18+
- [Rust](https://www.rust-lang.org/tools/install)（stable）
- **macOS**：Xcode Command Line Tools（`xcode-select --install`）
- **Windows**：[WebView2](https://developer.microsoft.com/microsoft-edge/webview2/)、MSVC 构建工具；打包 NSIS 安装包需在 Windows 主机或 CI 上执行

## 开发

先启动 API（及 Postgres），再开桌面壳：

```bash
# 仓库根目录
docker compose up -d postgres
go run ./cmd/server

# 另开终端
cd desktop
npm install
npm run dev
```

开发态加载 Vite（`http://127.0.0.1:5273`）。登录页可修改 API 地址（写入 localStorage）。

## 打包

```bash
cd desktop
npm install
npm run build          # 当前平台默认产物
npm run build:mac      # macOS: .app + .dmg
npm run build:windows  # Windows: NSIS setup（需在 Windows 上构建）
```

产物目录：`desktop/src-tauri/target/release/bundle/`。

未配置 Apple / Windows 代码签名时，安装包可本地安装；对外分发请按 [Tauri 分发文档](https://v2.tauri.app/distribute/) 配置签名与公证。

## 与 Web 的关系

| 场景 | API 地址 |
|---|---|
| 浏览器 + Vite 代理 | 相对路径 `/api`（不变） |
| 桌面打包产物 | 构建时注入 `VITE_API_BASE=http://127.0.0.1:8180`，可在登录页覆盖 |
