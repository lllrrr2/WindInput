# WindInput - Go 输入服务

这是 WindInput 输入法的 Go 服务端，负责处理拼音转换和词库管理。

## 功能

- 拼音音节解析
- 词库加载与查询
- 候选词生成与排序
- 命名管道 IPC 通信

## 构建

```bash
go build -o ../build/wind_input.exe ./cmd/service
```

## 运行

```bash
# 使用默认词库
./wind_input.exe

# 指定词库路径
./wind_input.exe -dict path/to/dict.txt

# 启用调试日志
./wind_input.exe -log debug
```

## 项目结构

```
wind_input/
├── cmd/service/           # 服务入口
├── internal/
│   ├── ipc/              # 命名管道通信
│   ├── engine/           # 输入引擎接口
│   │   └── pinyin/       # 拼音引擎实现
│   ├── dict/             # 词库管理
│   └── candidate/        # 候选词结构
└── go.mod
```

## 词库格式

词库文件采用简单的文本格式，每行一个词条：

```
拼音 汉字 权重
```

示例：
```
ni 你 100
hao 好 100
nihao 你好 150
```

- `#` 开头的行为注释
- 空行会被忽略
- 权重越高，候选词排序越靠前
