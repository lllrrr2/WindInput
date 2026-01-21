# WindInput - Windows 输入法

基于 Windows TSF 框架的输入法项目，使用 C++ 实现 TSF 核心层，Go 实现输入逻辑。

## 技术架构

- **C++ TSF 层**: 对接 Windows TSF API，处理输入法生命周期、按键事件
- **Go 输入服务**: 实现拼音转换、词库管理、候选词生成
- **通信方式**: 命名管道 (Named Pipe)

## 项目结构

```
wininput/
├── wind_tsf/                    # C++ TSF 核心
│   ├── src/               # 源代码
│   ├── include/           # 头文件
│   └── resource/          # 资源文件
├── wind_input/                    # Go 输入服务
│   ├── cmd/service/       # 服务入口
│   └── internal/          # 内部模块
├── dict/                  # 词库文件
├── installer/             # 安装脚本
├── build/                 # 构建输出
└── docs/                  # 文档
```

## 构建说明

### 构建 C++ DLL

```bash
cd wind_tsf
mkdir build && cd build
cmake ..
cmake --build . --config Release
```

### 构建 Go 服务

```bash
cd wind_input
go build -o ../build/wind_input.exe ./cmd/service
```

## 安装

以管理员权限运行：
```bash
installer\install.bat
```

## 卸载

以管理员权限运行：
```bash
installer\uninstall.bat
```

## 开发状态

- [x] 阶段 1: 项目初始化
- [ ] 阶段 2: MVP 输入功能
- [ ] 阶段 3: 注册与安装
- [ ] 阶段 4: 用户词库与优化

## 许可证

MIT
