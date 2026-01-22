# wind_tsf - C++ TSF 核心层

Windows TSF (Text Services Framework) 输入法核心实现，负责与 Windows 系统交互。

## 功能

- 实现 `ITfTextInputProcessor` 接口
- 处理按键事件 (`ITfKeyEventSink`)
- 语言栏图标显示 (`ITfLangBarItemButton`)
- 通过命名管道与 Go 服务通信
- 支持 Windows 10/11 现代输入框架

## 项目结构

```
wind_tsf/
├── src/
│   ├── dllmain.cpp           # DLL 入口点
│   ├── Globals.cpp           # 全局变量和 GUID 定义
│   ├── ClassFactory.cpp      # COM 类工厂
│   ├── TextService.cpp       # TSF 主服务实现
│   ├── KeyEventSink.cpp      # 按键事件处理
│   ├── LangBarItemButton.cpp # 语言栏图标
│   ├── IPCClient.cpp         # 命名管道客户端
│   └── Register.cpp          # TSF 注册/卸载
├── include/
│   ├── Globals.h
│   ├── TextService.h
│   ├── KeyEventSink.h
│   ├── LangBarItemButton.h
│   ├── IPCClient.h
│   ├── ClassFactory.h
│   └── Register.h
├── resource/
│   ├── resource.h            # 资源 ID 定义
│   └── wind_tsf.rc           # 资源文件
└── CMakeLists.txt
```

## 构建

需要：
- CMake 3.15+
- Visual Studio 2017+ (含 C++ 桌面开发工具)

```batch
mkdir build && cd build
cmake ..
cmake --build . --config Release
```

构建输出：`build/Release/wind_tsf.dll`

## 核心类说明

### CTextService

TSF 输入法主服务，实现以下接口：

| 接口 | 说明 |
|------|------|
| `ITfTextInputProcessor` | 输入法生命周期管理 |
| `ITfThreadMgrEventSink` | 线程管理器事件 |

关键方法：
- `Activate()` - 输入法激活，初始化所有组件
- `Deactivate()` - 输入法停用，释放资源
- `InsertText()` - 向应用程序插入文本
- `ToggleInputMode()` - 切换中英文模式

### CKeyEventSink

按键事件处理，实现 `ITfKeyEventSink` 接口：

- `OnTestKeyDown()` - 预判断是否处理该按键
- `OnKeyDown()` - 实际处理按键
- 自动识别 Ctrl/Alt 组合键并放行

处理的按键：
- `A-Z` - 拼音输入
- `1-9` - 选择候选词
- `Space` - 选择第一个候选词
- `Enter` - 提交原始拼音
- `Escape` - 取消输入
- `Backspace` - 删除最后一个字符
- `Shift` - 切换中英文模式

### CLangBarItemButton

语言栏图标，显示当前中/英文状态：

- 使用 `GUID_LBI_INPUTMODE` 确保在 Windows 11 输入指示器中显示
- 支持点击切换模式
- 图标自动更新

### CIPCClient

命名管道客户端：

- 管道名称: `\\.\pipe\wind_input`
- 协议: 长度前缀 + JSON
- 自动重连机制
- 自动启动 Go 服务

## TSF 注册

### 必需的分类 GUID

```cpp
GUID_TFCAT_TIP_KEYBOARD           // 键盘类输入法
GUID_TFCAT_TIPCAP_IMMERSIVESUPPORT // UWP 应用支持 (Windows 8+)
GUID_TFCAT_TIPCAP_SYSTRAYSUPPORT   // 系统托盘支持 (Windows 8+)
GUID_TFCAT_TIPCAP_UIELEMENTENABLED // UI 元素支持 (Windows 8+)
```

### 注册命令

```batch
# 注册
regsvr32 wind_tsf.dll

# 卸载
regsvr32 /u wind_tsf.dll
```

需要管理员权限。

## GUID 说明

项目中定义的 GUID：

| GUID | 用途 |
|------|------|
| `c_clsidTextService` | COM 类 ID |
| `c_guidProfile` | 语言配置文件 ID |
| `GUID_LBI_INPUTMODE` | 语言栏图标 ID |

如需部署，应生成新的唯一 GUID：
```powershell
[guid]::NewGuid()
```

## 调试

1. 构建 Debug 配置:
   ```batch
   cmake --build . --config Debug
   ```

2. 注册 Debug DLL

3. 打开 DebugView 或 Visual Studio 输出窗口

4. 附加到任意 TSF 应用进程 (如 notepad.exe)

所有调试输出使用 `OutputDebugStringW`，前缀为 `[WindInput]`。

## IPC 消息格式

### 发送给 Go 服务

```json
// 按键事件
{"type": "key_event", "data": {"key": "a", "keycode": 65, "modifiers": 0}}

// 光标位置
{"type": "caret_update", "data": {"x": 100, "y": 200, "height": 20}}

// 焦点丢失
{"type": "focus_lost"}

// 切换模式
{"type": "toggle_mode"}
```

### 从 Go 服务接收

```json
{"type": "ack"}
{"type": "insert_text", "data": {"text": "你好"}}
{"type": "mode_changed", "data": {"chinese_mode": true}}
{"type": "clear_composition"}
```

## 注意事项

- 修改 DLL 后需要卸载再重新注册
- 卸载前确保所有应用已切换到其他输入法
- 调试时可能需要结束 `ctfmon.exe` 进程
- Windows 11 要求使用 `ITfInputProcessorProfileMgr` 注册
