# wind_input - Go 输入服务

WindInput 输入法的 Go 服务端，负责输入逻辑处理、词库管理和候选窗口显示。

## 功能

- 拼音输入引擎
- 词库加载与查询
- 候选词生成与排序
- 命名管道 IPC 服务
- 原生 Windows 候选窗口
- YAML 配置系统
- DPI 缩放支持

## 项目结构

```
wind_input/
├── cmd/service/
│   └── main.go              # 服务入口
└── internal/
    ├── bridge/              # C++ 通信层
    │   ├── protocol.go      # IPC 协议定义
    │   └── server.go        # 命名管道服务端
    │
    ├── coordinator/         # 输入协调器
    │   └── coordinator.go   # 状态管理、业务逻辑
    │
    ├── engine/              # 输入引擎
    │   ├── engine.go        # 引擎接口
    │   └── pinyin/
    │       ├── pinyin.go    # 拼音引擎
    │       └── syllable.go  # 音节解析
    │
    ├── dict/                # 词库
    │   ├── dict.go          # 词库接口
    │   └── loader.go        # 词库加载器
    │
    ├── candidate/           # 候选词
    │   └── candidate.go     # 候选词结构
    │
    ├── ui/                  # 候选窗口 UI
    │   ├── manager.go       # UI 管理器
    │   ├── window.go        # 窗口操作 (Win32)
    │   ├── renderer.go      # 渲染器 (Go image)
    │   └── protocol.go      # UI 数据结构
    │
    └── config/              # 配置系统
        └── config.go        # 配置加载/保存
```

## 构建

需要 Go 1.21+：

```bash
go build -o ../build/wind_input.exe ./cmd/service

# 构建 Windows GUI 应用 (无控制台窗口)
go build -ldflags "-H windowsgui" -o ../build/wind_input.exe ./cmd/service
```

## 运行

```bash
# 默认运行
./wind_input.exe

# 指定词库
./wind_input.exe -dict path/to/dict.txt

# 调试模式
./wind_input.exe -log debug

# 保存默认配置
./wind_input.exe -save-config
```

## 核心组件

### Bridge Server

处理与 C++ TSF 的通信：

```go
type Server struct {
    handler MessageHandler
}

type MessageHandler interface {
    HandleKeyEvent(data KeyEventData) *KeyEventResult
    HandleCaretUpdate(data CaretData) error
    HandleFocusLost()
    HandleToggleMode() bool
}
```

命名管道：`\\.\pipe\wind_input`

### Coordinator

输入状态协调器，核心业务逻辑：

```go
type Coordinator struct {
    engine    engine.Engine
    uiManager *ui.Manager

    chineseMode bool       // 中/英文模式
    inputBuffer string     // 当前拼音输入
    candidates  []Candidate
    currentPage int
}
```

状态流转：
- 英文模式：直接透传字母
- 中文模式：累积拼音 → 查询候选词 → 显示候选窗口
- Shift：切换中英文

### Engine

输入引擎接口：

```go
type Engine interface {
    Convert(input string, maxCandidates int) ([]Candidate, error)
    Reset()
}
```

拼音引擎实现：
- 音节解析: `nihao` → `["ni", "hao"]`
- 词组匹配: 优先匹配长词
- 权重排序: 高频词优先

### UI Manager

候选窗口管理：

```go
type Manager struct {
    window   *CandidateWindow
    renderer *Renderer
    cmdCh    chan UICommand  // 异步命令队列
}
```

特点：
- 异步更新，不阻塞 IPC
- 使用 Go image 渲染
- 调用 Win32 API 显示窗口
- 自动 DPI 缩放

### Config

配置系统：

```go
type Config struct {
    General    GeneralConfig    `yaml:"general"`
    Dictionary DictionaryConfig `yaml:"dictionary"`
    Hotkeys    HotkeyConfig     `yaml:"hotkeys"`
    UI         UIConfig         `yaml:"ui"`
}
```

配置路径：`%APPDATA%\WindInput\config.yaml`

## IPC 协议

### 请求 (C++ → Go)

| 类型 | 说明 |
|------|------|
| `key_event` | 按键事件 |
| `caret_update` | 光标位置更新 |
| `focus_lost` | 焦点丢失 |
| `toggle_mode` | 请求切换模式 |

### 响应 (Go → C++)

| 类型 | 说明 |
|------|------|
| `ack` | 确认，无操作 |
| `insert_text` | 插入文字 |
| `clear_composition` | 清除组字 |
| `mode_changed` | 模式已切换 |

## 词库格式

纯文本格式，每行一个词条：

```
拼音 汉字 权重
```

示例 (`dict/pinyin/base.txt`)：
```
# 这是注释
ni 你 100
ni 泥 20
hao 好 100
nihao 你好 150
zhongguo 中国 150
```

- `#` 开头为注释
- 空行忽略
- 权重越高排序越靠前

## 配置文件

`%APPDATA%\WindInput\config.yaml`：

```yaml
general:
  start_in_chinese_mode: true   # 启动默认中文
  log_level: info               # debug/info/warn/error

dictionary:
  system_dict: dict/pinyin/base.txt
  user_dict: user_dict.txt

hotkeys:
  toggle_mode: shift            # 中英切换键

ui:
  font_size: 18                 # 字体大小
  candidates_per_page: 9        # 每页候选数
  font_path: ""                 # 自定义字体路径
```

## 依赖

```
golang.org/x/sys/windows  # Windows API
golang.org/x/image/font   # 字体渲染
gopkg.in/yaml.v3          # YAML 解析
```

## 调试

启用详细日志：
```bash
./wind_input.exe -log debug
```

日志输出到 stdout，使用 `slog` 结构化日志：
```
2024/01/22 10:00:00 INFO HandleKeyEvent key=a keycode=65 modifiers=0
2024/01/22 10:00:00 INFO Input buffer updated buffer=a
2024/01/22 10:00:00 INFO Got candidates count=5
```

## 单例控制

服务使用命名互斥体确保单例运行：
- 互斥体名称: `Global\WindInputIMEService`
- 如果已有实例运行，新实例会弹出提示并退出

## 扩展

### 添加新引擎

1. 实现 `Engine` 接口：

```go
type WubiEngine struct {
    dict dict.Dict
}

func (e *WubiEngine) Convert(input string, max int) ([]Candidate, error) {
    // 五笔编码逻辑
}

func (e *WubiEngine) Reset() {}
```

2. 在 `main.go` 中注册

### 添加新消息类型

1. 在 `bridge/protocol.go` 定义
2. 在 `bridge/server.go` 处理
3. 在 `MessageHandler` 接口添加方法
4. 在 `Coordinator` 实现

## 注意事项

- 服务需要在 C++ DLL 加载前启动
- 候选窗口渲染使用 Go 原生图像库
- Windows 消息循环运行在独立协程
- IPC 处理和 UI 更新是异步的
