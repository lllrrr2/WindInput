# WindInput - Windows 输入法

基于 Windows TSF (Text Services Framework) 框架的中文输入法，采用 C++ 实现 TSF 核心层，Go 实现输入引擎和候选窗口。

## 特性

- **TSF 原生集成**: 完全兼容 Windows 10/11 的输入法框架
- **C++/Go 混合架构**: C++ 处理系统接口，Go 处理输入逻辑
- **命名管道通信**: 低延迟的进程间通信
- **拼音输入**: 支持全拼输入，可扩展支持五笔等
- **中英文切换**: Shift 键切换，任务栏图标同步显示
- **候选窗口**: 跟随光标的原生 Windows 候选窗口
- **高 DPI 支持**: 自动适配高分辨率显示器
- **配置文件**: YAML 格式配置，支持自定义

## 系统要求

- Windows 10 或 Windows 11
- Visual Studio 2017 或更高版本 (含 C++ 桌面开发工具)
- Go 1.21 或更高版本
- CMake 3.15 或更高版本

## 快速开始

### 构建项目

使用一键构建脚本：
```batch
build_all.bat
```

或手动分步构建：

```batch
# 构建 C++ TSF DLL
cd wind_tsf
mkdir build && cd build
cmake ..
cmake --build . --config Release

# 构建 Go 服务
cd wind_input
go build -o ../build/wind_input.exe ./cmd/service
```

### 安装

以**管理员权限**运行：
```batch
installer\install.bat
```

### 卸载

以**管理员权限**运行：
```batch
installer\uninstall.bat
```

## 项目结构

```
tsfdemo/
├── wind_tsf/              # C++ TSF 核心 (DLL)
│   ├── src/               # 源代码
│   │   ├── dllmain.cpp    # DLL 入口点
│   │   ├── ClassFactory.cpp
│   │   ├── TextService.cpp # TSF 主服务
│   │   ├── KeyEventSink.cpp # 按键处理
│   │   ├── LangBarItemButton.cpp # 语言栏图标
│   │   ├── IPCClient.cpp  # 命名管道客户端
│   │   └── Register.cpp   # TSF 注册
│   ├── include/           # 头文件
│   └── resource/          # 资源文件 (图标等)
│
├── wind_input/            # Go 输入服务
│   ├── cmd/service/       # 服务入口
│   └── internal/
│       ├── bridge/        # 与 C++ 的 IPC 通信
│       ├── coordinator/   # 输入协调器
│       ├── engine/        # 输入引擎接口
│       │   └── pinyin/    # 拼音引擎
│       ├── dict/          # 词库管理
│       ├── ui/            # 候选窗口 UI
│       └── config/        # 配置管理
│
├── wind_setting/          # 设置工具 (待开发)
│
├── dict/                  # 词库文件
│   └── pinyin/            # 拼音词库
│       └── base.txt       # 基础词库
│
├── installer/             # 安装脚本
│   ├── install.bat
│   └── uninstall.bat
│
├── build/                 # 构建输出
└── docs/                  # 开发文档
```

## 技术架构

```
┌──────────────────────────────────────────────────────────┐
│                    Windows 应用程序                       │
│                (记事本、浏览器、Office 等)                 │
└────────────────────────┬─────────────────────────────────┘
                         │ TSF 接口
┌────────────────────────┼─────────────────────────────────┐
│    wind_tsf.dll        │           C++                   │
│   ┌────────────────────▼───────────────────────┐         │
│   │              TextService                   │         │
│   │     ITfTextInputProcessor 实现             │         │
│   └─────────────┬───────────────────┬──────────┘         │
│                 │                   │                    │
│   ┌─────────────▼─────┐   ┌────────▼────────┐           │
│   │  KeyEventSink     │   │ LangBarItemButton│           │
│   │  按键事件处理      │   │   语言栏图标    │           │
│   └─────────────┬─────┘   └─────────────────┘           │
│                 │                                        │
│   ┌─────────────▼─────────────────────────┐             │
│   │           IPCClient                    │             │
│   │        命名管道客户端                   │             │
│   └─────────────┬─────────────────────────┘             │
└─────────────────┼────────────────────────────────────────┘
                  │ Named Pipe: \\.\pipe\wind_input
┌─────────────────┼────────────────────────────────────────┐
│  wind_input.exe │           Go                           │
│   ┌─────────────▼─────────────────────────┐             │
│   │          Bridge Server                 │             │
│   │         命名管道服务端                  │             │
│   └─────────────┬─────────────────────────┘             │
│                 │                                        │
│   ┌─────────────▼─────────────────────────┐             │
│   │          Coordinator                   │             │
│   │     输入协调器 (状态管理)              │             │
│   └─────┬───────────────────┬─────────────┘             │
│         │                   │                            │
│   ┌─────▼─────┐       ┌─────▼──────────┐                │
│   │  Engine   │       │   UI Manager   │                │
│   │ 拼音引擎  │       │   候选窗口     │                │
│   └─────┬─────┘       └────────────────┘                │
│         │                                                │
│   ┌─────▼─────┐                                         │
│   │   Dict    │                                         │
│   │   词库    │                                         │
│   └───────────┘                                         │
└──────────────────────────────────────────────────────────┘
```

## 配置

配置文件位于 `%APPDATA%\WindInput\config.yaml`:

```yaml
general:
  start_in_chinese_mode: true  # 启动时默认中文模式
  log_level: info              # 日志级别: debug, info, warn, error

dictionary:
  system_dict: dict/pinyin/base.txt
  user_dict: user_dict.txt

hotkeys:
  toggle_mode: shift           # 中英切换键

ui:
  font_size: 18                # 候选窗口字体大小
  candidates_per_page: 9       # 每页候选词数量
```

## 使用方法

1. 安装后，使用 `Win + Space` 或 `Ctrl + Shift` 切换到 WindInput 输入法
2. 输入拼音，候选窗口自动显示
3. 使用数字键 1-9 选择候选词，或空格选择第一个
4. 按 `Shift` 切换中英文模式
5. 按 `Esc` 取消当前输入
6. 按 `Enter` 输出原始拼音

## 开发状态

- [x] 项目初始化与基础框架
- [x] 命名管道 IPC 通信
- [x] TSF 注册与 Windows 10/11 兼容
- [x] 拼音引擎基础实现
- [x] 按键事件处理
- [x] 候选窗口显示
- [x] 中英文切换
- [x] 语言栏图标 (Windows 11 兼容)
- [x] 配置系统
- [x] DPI 缩放支持
- [ ] 用户词库学习
- [ ] 模糊音支持
- [ ] 设置界面

## 文档

详细开发文档请参阅 [docs/architecture.md](docs/architecture.md)

## 许可证

MIT License

## 致谢

- [Windows TSF 官方文档](https://docs.microsoft.com/en-us/windows/win32/tsf/text-services-framework)
- [Windows Classic Samples](https://github.com/microsoft/Windows-classic-samples)
