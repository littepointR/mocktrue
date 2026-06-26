# PortWeave

[![CI](https://github.com/littepointR/portweave/actions/workflows/ci.yml/badge.svg)](https://github.com/littepointR/portweave/actions/workflows/ci.yml)

PortWeave 是一个跨平台高性能嵌入式调试工具，面向串口、协议调试、虚拟设备模拟和自动化验证场景。

PortWeave is a cross-platform embedded debugging toolkit for serial communication, protocol debugging, virtual device simulation, and automated validation workflows.

## 功能特性 / Features

- 串口拓扑图：通过节点和连接线组织物理串口、虚拟串口、桥接、监控、分流、过滤器、发送器和接收器。
- 自动收发与演示工作区：示例配置基于真实功能构建，可使用虚拟串口循环运行数据，并内置串口过滤与日志演示。
- 串口监控：自动创建虚拟监听端口，按实际时间记录收发帧。
- Modbus 支持：支持 Modbus RTU/ASCII 主站、从站、多 Unit ID、寄存器表格和原始帧显示。
- FECbus 支持：支持 FECbus 主从站、多从站、数据帧分段标注和自定义功能码。
- 脚本节点：提供受限 PortWeave 脚本 API，用于生成、转换和分析串口数据。
- MCP 服务：暴露串口和运行时能力，便于外部工具和自动化流程集成；串口拓扑 MCP 工具提供 provider catalog、校验/启动/缓冲区查询，以及只读 demo template（包含过滤器与日志示例）。
- 配置工作区：每个标签页可独立保存、加载和恢复配置。

## 技术栈 / Tech Stack

- Desktop: Wails v3
- Backend: Go
- Frontend: Vue 3, TypeScript, Vite, Naive UI, Pinia, Monaco Editor
- Tests: Vitest, Playwright, Go test
- CI: GitHub Actions

## 环境要求 / Requirements

- Go `1.26.0`，以 `go.mod` 为准
- Node.js `22`，与 CI 配置一致
- pnpm `10.32.1`
- Wails CLI v3 alpha

安装 Wails CLI：

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha2.103
```

安装前端依赖：

```bash
cd frontend
pnpm install
```

## 开发 / Development

从仓库根目录启动桌面应用：

```bash
wails3 dev -config ./build/config.yml -port 9245
```

也可以使用 Taskfile：

```bash
wails3 task dev
```

构建和打包：

```bash
wails3 task build
wails3 task package
```

运行服务端模式：

```bash
wails3 task run:server
```

## 测试 / Tests

前端测试：

```bash
cd frontend
pnpm test -- --run
pnpm exec vue-tsc --noEmit
pnpm run build:dev
```

后端串口模块测试：

```bash
go test ./internal/modules/serial/... -count=1
```

完整 Go 测试：

```bash
go test ./... -count=1
```

Windows 虚拟 COM 集成测试是可选流程，详见 [Windows Serial Testing](docs/development/windows-serial-testing.md)。

## 项目结构 / Project Structure

```text
.
├── build/                 # Wails build configuration and platform tasks
├── docs/                  # Design notes and planning documents
├── frontend/              # Vue/TypeScript frontend
├── internal/modules/      # Backend modules, including serial, Modbus, FECbus, MCP
├── main.go                # Application entry point
├── Taskfile.yml           # Development, build, package, and server tasks
└── README.md
```

## 贡献 / Contributing

请先阅读 [CONTRIBUTING.md](CONTRIBUTING.md)。提交变更前至少运行与改动相关的测试；涉及前端或后端公共能力时，同时运行类型检查和构建检查。

安全问题请按 [SECURITY.md](SECURITY.md) 报告，不要在公开 issue 中披露敏感细节。

## 许可证 / License

PortWeave is licensed under the [MIT License](LICENSE).
