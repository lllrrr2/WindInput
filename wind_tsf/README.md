# WindInput - C++ TSF 核心

这是 WindInput 输入法的 C++ 核心层，实现 Windows TSF (Text Services Framework) 接口。

## 功能

- TSF Text Input Processor 实现
- COM 组件注册与卸载
- 按键事件处理
- 命名管道 IPC 客户端

## 构建

需要 CMake 3.15+ 和 Visual Studio 2017+

```bash
cd wind_tsf
mkdir build
cd build
cmake ..
cmake --build . --config Release
```

## 项目结构

```
wind_tsf/
├── src/                   # 源代码
│   ├── dllmain.cpp       # DLL 入口点
│   ├── Globals.cpp       # 全局变量和 GUID
│   ├── ClassFactory.cpp  # COM 类工厂
│   ├── TextService.cpp   # TSF Text Service 主实现
│   ├── KeyEventSink.cpp  # 按键事件处理
│   ├── IPCClient.cpp     # 命名管道客户端
│   └── Register.cpp      # 注册/卸载
├── include/              # 公共头文件
├── resource/             # 资源文件
└── CMakeLists.txt
```

## GUID 说明

项目中使用的 GUID 需要在实际部署时替换为唯一值。可以使用以下工具生成：

- Visual Studio: Tools -> Create GUID
- PowerShell: `[guid]::NewGuid()`
- 命令行: `uuidgen`

需要替换的 GUID：
- `c_clsidTextService` - Text Service CLSID
- `c_guidProfile` - Language Profile GUID
- `c_guidLangBarItemButton` - Language Bar Button GUID

## 调试

使用 Visual Studio 调试 TSF 组件：

1. 构建 Debug 配置
2. 注册 DLL：`regsvr32 wind_tsf.dll`
3. 附加到进程：任意支持 TSF 的应用（如记事本）
4. 设置断点并调试

## 注意事项

- 修改代码后需要重新注册 DLL
- 卸载前确保没有应用正在使用输入法
- 调试时可能需要管理员权限
