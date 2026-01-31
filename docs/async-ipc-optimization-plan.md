# IPC 异步优化实施计划

## 目标

1. 消除 IPC 同步调用导致的卡顿
2. 解决快速输入时状态漂移问题
3. 支持多 TSF 客户端状态同步

## 架构设计

### 1. 状态同步模型

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Go 服务 (权威状态源)                           │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Coordinator                                                     │   │
│  │  - chineseMode: bool (权威)                                      │   │
│  │  - fullWidth: bool (权威)                                        │   │
│  │  - chinesePunctuation: bool (权威)                               │   │
│  │  - hotkeyList: []uint32 (权威)                                   │   │
│  │  - inputBuffer: string                                           │   │
│  │  - candidates: []Candidate                                       │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
                              ↑↓ IPC
     ┌────────────────────────┼────────────────────────┐
     ↓                        ↓                        ↓
┌─────────────┐        ┌─────────────┐        ┌─────────────┐
│ TSF Client 1│        │ TSF Client 2│        │ TSF Client N│
│ (Notepad)   │        │ (VSCode)    │        │ (Browser)   │
│             │        │             │        │             │
│ localState: │        │ localState: │        │ localState: │
│ - cached    │        │ - cached    │        │ - cached    │
│   modeCopy  │        │   modeCopy  │        │   modeCopy  │
└─────────────┘        └─────────────┘        └─────────────┘
```

### 2. 状态同步规则

1. **Go 端是权威状态源**：所有模式切换最终由 Go 端决定
2. **TSF 端缓存状态**：用于快速决策（如热键匹配、按键分类）
3. **激活时同步**：TSF 发送 `IME_ACTIVATED`，Go 返回完整状态
4. **模式变更时广播**：Go 端模式变更后，主动推送给所有连接的 TSF 客户端

### 3. 事件分类与处理策略

| 事件类型 | 处理策略 | 是否等待响应 | 说明 |
|----------|----------|--------------|------|
| 普通字符输入 | 异步发送 | 否 | Go 处理后推送候选词更新 |
| 提交请求 (Space/Enter/数字) | Barrier | 是（异步等待） | 等待 COMMIT_RESULT 后提交 |
| 模式切换键 | 同步（短超时） | 是（10ms） | 需要立即知道新模式 |
| 热键 | 本地匹配 + 异步通知 | 否 | 热键清单预下发 |
| 光标更新 | 异步发送 | 否 | 已实现 |
| 焦点变更 | 根据类型 | 视情况 | 获得焦点需同步状态 |

---

## 阶段一：协议扩展与状态机

### 1.1 扩展 KeyPayload 结构

```cpp
// BinaryProtocol.h
struct KeyPayload
{
    uint32_t keyCode;      // Virtual key code
    uint32_t scanCode;     // Scan code
    uint32_t modifiers;    // Modifier flags (snapshot at event time)
    uint8_t  eventType;    // 0=KeyDown, 1=KeyUp
    uint8_t  toggles;      // bit0:CapsLock, bit1:NumLock, bit2:ScrollLock
    uint16_t eventSeq;     // Monotonic sequence number
};
static_assert(sizeof(KeyPayload) == 16, "KeyPayload must be 16 bytes");

// Toggle flags
constexpr uint8_t TOGGLE_CAPSLOCK   = 0x01;
constexpr uint8_t TOGGLE_NUMLOCK    = 0x02;
constexpr uint8_t TOGGLE_SCROLLLOCK = 0x04;
```

### 1.2 TSF 端状态机

```cpp
// KeyEventSink.h
class CKeyEventSink : public ITfKeyEventSink
{
private:
    // 修饰键状态机（由 KeyDown/KeyUp 事件驱动，不依赖 GetAsyncKeyState）
    uint32_t _modsState = 0;

    // 事件序号
    uint16_t _eventSeq = 0;

    // 从 Go 同步的缓存状态
    bool _cachedChineseMode = true;
    bool _cachedFullWidth = false;
    bool _cachedChinesePunct = true;

    // 状态机更新
    void _UpdateModsOnKeyDown(WPARAM vk);
    void _UpdateModsOnKeyUp(WPARAM vk);
    uint32_t _GetModsSnapshot() const { return _modsState; }
    uint8_t _GetTogglesSnapshot() const;

    // 状态同步
    void _SyncStateFromGo(const StatusPayload& status);
};
```

### 1.3 状态机实现

```cpp
// KeyEventSink.cpp
void CKeyEventSink::_UpdateModsOnKeyDown(WPARAM vk)
{
    switch (vk)
    {
    case VK_SHIFT:
    case VK_LSHIFT:
        _modsState |= (KEYMOD_SHIFT | KEYMOD_LSHIFT);
        break;
    case VK_RSHIFT:
        _modsState |= (KEYMOD_SHIFT | KEYMOD_RSHIFT);
        break;
    case VK_CONTROL:
    case VK_LCONTROL:
        _modsState |= (KEYMOD_CTRL | KEYMOD_LCTRL);
        break;
    case VK_RCONTROL:
        _modsState |= (KEYMOD_CTRL | KEYMOD_RCTRL);
        break;
    case VK_MENU:
    case VK_LMENU:
        _modsState |= KEYMOD_ALT;
        break;
    case VK_RMENU:
        _modsState |= KEYMOD_ALT;  // Note: may need special AltGr handling
        break;
    case VK_LWIN:
    case VK_RWIN:
        _modsState |= KEYMOD_WIN;
        break;
    }
}

void CKeyEventSink::_UpdateModsOnKeyUp(WPARAM vk)
{
    switch (vk)
    {
    case VK_SHIFT:
        // Clear both generic and specific if no shift keys are held
        if (!(GetAsyncKeyState(VK_LSHIFT) & 0x8000) &&
            !(GetAsyncKeyState(VK_RSHIFT) & 0x8000))
        {
            _modsState &= ~(KEYMOD_SHIFT | KEYMOD_LSHIFT | KEYMOD_RSHIFT);
        }
        break;
    case VK_LSHIFT:
        _modsState &= ~KEYMOD_LSHIFT;
        if (!(_modsState & KEYMOD_RSHIFT))
            _modsState &= ~KEYMOD_SHIFT;
        break;
    case VK_RSHIFT:
        _modsState &= ~KEYMOD_RSHIFT;
        if (!(_modsState & KEYMOD_LSHIFT))
            _modsState &= ~KEYMOD_SHIFT;
        break;
    // ... similar for Ctrl, Alt, Win
    }
}

uint8_t CKeyEventSink::_GetTogglesSnapshot() const
{
    uint8_t toggles = 0;
    if (GetKeyState(VK_CAPITAL) & 0x01) toggles |= TOGGLE_CAPSLOCK;
    if (GetKeyState(VK_NUMLOCK) & 0x01) toggles |= TOGGLE_NUMLOCK;
    if (GetKeyState(VK_SCROLL) & 0x01)  toggles |= TOGGLE_SCROLLLOCK;
    return toggles;
}
```

### 1.4 Go 端对应更新

```go
// binary_protocol.go
type KeyPayload struct {
    KeyCode   uint32
    ScanCode  uint32
    Modifiers uint32
    EventType uint8
    Toggles   uint8   // NEW: bit0:CapsLock, bit1:NumLock, bit2:ScrollLock
    EventSeq  uint16  // NEW: monotonic sequence number
}

const (
    ToggleCapsLock   = 0x01
    ToggleNumLock    = 0x02
    ToggleScrollLock = 0x04
)
```

---

## 阶段二：Barrier 机制

### 2.1 新增协议命令

```cpp
// BinaryProtocol.h - 新增命令
constexpr uint16_t CMD_COMMIT_REQUEST  = 0x0104; // C++ -> Go: 请求提交
constexpr uint16_t CMD_COMMIT_RESULT   = 0x0105; // Go -> C++: 提交结果（推送）
constexpr uint16_t CMD_STATE_PUSH      = 0x0206; // Go -> C++: 状态推送（广播）

// Commit request payload
struct CommitRequestPayload
{
    uint16_t barrierSeq;     // Barrier sequence number
    uint16_t triggerKey;     // VK code that triggered commit (Space/Enter/1-9)
    uint32_t modifiers;      // Modifier state at trigger time
    // Followed by UTF-8 input buffer (length = header.length - 8)
};
static_assert(sizeof(CommitRequestPayload) == 8, "CommitRequestPayload must be 8 bytes");

// Commit result payload (pushed from Go)
struct CommitResultPayload
{
    uint16_t barrierSeq;     // Matching barrier sequence
    uint16_t flags;          // bit0: modeChanged, bit1: hasNewComposition
    uint32_t textLength;     // Length of commit text
    uint32_t compositionLength; // Length of new composition (0 if none)
    // Followed by UTF-8 text, then optional new composition
};
static_assert(sizeof(CommitResultPayload) == 12, "CommitResultPayload must be 12 bytes");
```

### 2.2 TSF 端 Barrier 管理

```cpp
// KeyEventSink.h
struct PendingBarrier
{
    uint16_t barrierSeq;
    std::wstring composition;  // Composition at request time
    DWORD requestTime;         // GetTickCount() at request
    bool waiting;
};

class CKeyEventSink
{
private:
    uint16_t _nextBarrierSeq = 1;
    PendingBarrier _pendingCommit = {0, L"", 0, false};

    // Barrier 超时（如果 Go 没响应，fallback 处理）
    static constexpr DWORD BARRIER_TIMEOUT_MS = 500;

    bool _SendCommitRequest(uint16_t barrierSeq, uint16_t triggerKey, uint32_t mods, const std::string& inputBuffer);
    void _HandleCommitResult(const CommitResultPayload& result, const std::wstring& text, const std::wstring& newComp);
    void _CheckBarrierTimeout();
};
```

### 2.3 提交请求流程

```cpp
// KeyEventSink.cpp
STDAPI CKeyEventSink::OnKeyDown(ITfContext* pContext, WPARAM wParam, LPARAM lParam, BOOL* pfEaten)
{
    // ... existing checks ...

    // Check if this is a commit trigger key
    bool isCommitKey = (wParam == VK_SPACE || wParam == VK_RETURN ||
                        (wParam >= '1' && wParam <= '9'));

    if (isCommitKey && _isComposing)
    {
        // 1. Generate barrier sequence
        uint16_t barrierSeq = _nextBarrierSeq++;

        // 2. Send async commit request
        if (_SendCommitRequest(barrierSeq, (uint16_t)wParam, _GetModsSnapshot(), _inputBuffer))
        {
            // 3. Mark as waiting (don't commit yet)
            _pendingCommit = {
                barrierSeq,
                _currentComposition,
                GetTickCount(),
                true
            };

            // 4. Consume the key immediately (prevent leaking to app)
            *pfEaten = TRUE;
            return S_OK;
        }
        else
        {
            // Fallback: if IPC fails, use existing sync path
            // ... existing code ...
        }
    }

    // ... rest of OnKeyDown ...
}
```

### 2.4 Go 端处理提交请求

```go
// coordinator.go
func (c *Coordinator) HandleCommitRequest(req bridge.CommitRequest) *bridge.CommitResult {
    c.mu.Lock()
    defer c.mu.Unlock()

    var commitText string
    var newComposition string
    var modeChanged bool

    switch req.TriggerKey {
    case 0x20: // VK_SPACE
        result := c.handleSpaceInternal()
        if result != nil {
            commitText = result.Text
            modeChanged = result.ModeChanged
            newComposition = result.NewComposition
        }
    case 0x0D: // VK_RETURN
        result := c.handleEnterInternal()
        if result != nil {
            commitText = result.Text
        }
    default:
        if req.TriggerKey >= '1' && req.TriggerKey <= '9' {
            index := int(req.TriggerKey - '1')
            result := c.selectCandidateInternal(index)
            if result != nil {
                commitText = result.Text
            }
        }
    }

    return &bridge.CommitResult{
        BarrierSeq:     req.BarrierSeq,
        Text:           commitText,
        NewComposition: newComposition,
        ModeChanged:    modeChanged,
    }
}
```

### 2.5 推送机制（Go -> TSF）

```go
// bridge/server.go
type Server struct {
    // ... existing fields ...

    // 活跃客户端连接（用于推送）
    mu            sync.RWMutex
    activeClients map[windows.Handle]*clientConn
}

type clientConn struct {
    handle   windows.Handle
    writer   io.Writer
    clientID int
}

// PushToAllClients 向所有活跃客户端推送消息
func (s *Server) PushToAllClients(data []byte) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    for _, client := range s.activeClients {
        // Non-blocking write with timeout
        go s.pushToClient(client, data)
    }
}

// PushCommitResult 推送提交结果
func (s *Server) PushCommitResult(result *CommitResult) {
    data := s.codec.EncodeCommitResult(result)
    // Only push to the client that made the request
    // (We need to track which client is in composing state)
}
```

### 2.6 TSF 端接收推送

```cpp
// IPCClient.cpp
// 需要改为双工模式，或使用单独的推送通道

// 方案 A：在现有管道上轮询
bool CIPCClient::PollPushMessages(std::vector<PushMessage>& messages)
{
    // Non-blocking check for incoming push messages
    DWORD available = 0;
    if (!PeekNamedPipe(_hPipe, nullptr, 0, nullptr, &available, nullptr))
        return false;

    if (available == 0)
        return true;  // No messages

    // Read and parse push messages
    // ...
}

// 方案 B：单独的推送通道（推荐）
// 创建第二个命名管道专门用于 Go -> TSF 的推送
```

---

## 阶段三：普通字符异步化

### 3.1 异步发送字符输入

```cpp
// KeyEventSink.cpp
if (isLetterKey && _cachedChineseMode && !(_modsSnapshot & (KEYMOD_CTRL | KEYMOD_ALT)))
{
    // 1. Update local state machine
    _UpdateModsOnKeyDown(wParam);

    // 2. Send async key event (don't wait for response)
    KeyPayload payload;
    payload.keyCode = (uint32_t)wParam;
    payload.scanCode = LOBYTE(HIWORD(lParam));
    payload.modifiers = _GetModsSnapshot();
    payload.eventType = KEY_EVENT_DOWN;
    payload.toggles = _GetTogglesSnapshot();
    payload.eventSeq = _eventSeq++;

    _pIPCClient->SendKeyEventAsync(payload);  // Non-blocking

    // 3. Consume the key immediately
    *pfEaten = TRUE;
    return S_OK;
}
```

### 3.2 Go 端推送候选词更新

```go
// coordinator.go
func (c *Coordinator) HandleKeyEventAsync(data bridge.KeyEventData) {
    c.mu.Lock()
    result := c.processKeyEventInternal(data)
    c.mu.Unlock()

    // If candidates updated, push to TSF
    if result != nil && result.Type == bridge.ResponseTypeUpdateComposition {
        c.pushCandidatesUpdate(result)
    }
}
```

---

## 阶段四：多客户端状态广播

### 4.1 状态变更广播

```go
// coordinator.go
func (c *Coordinator) broadcastStateChange() {
    status := &bridge.StatusUpdateData{
        ChineseMode:        c.chineseMode,
        FullWidth:          c.fullWidth,
        ChinesePunctuation: c.chinesePunctuation,
        ToolbarVisible:     c.toolbarVisible,
        CapsLock:           ui.GetCapsLockState(),
    }

    // Don't include hotkeys in broadcast (they don't change frequently)
    c.bridgeServer.PushStateToAllClients(status)
}

func (c *Coordinator) handleToolbarToggleMode() {
    c.mu.Lock()
    c.chineseMode = !c.chineseMode
    // ... other logic ...
    c.mu.Unlock()

    // Broadcast to all TSF clients
    c.broadcastStateChange()
}
```

### 4.2 TSF 端处理状态推送

```cpp
// TextService.cpp
void CTextService::HandleStatePush(const StatusPayload& status)
{
    // Update cached state
    _pKeyEventSink->SyncStateFromGo(status);

    // Update language bar indicator
    if (_pLangBarButton)
    {
        _pLangBarButton->UpdateState(status.IsChineseMode());
    }
}
```

---

## 实施顺序与优先级

| 阶段 | 任务 | 优先级 | 估计工作量 | 依赖 |
|------|------|--------|------------|------|
| 1.1 | KeyPayload 扩展（eventSeq, toggles） | P0 | 2h | - |
| 1.2 | TSF 状态机实现 | P0 | 4h | 1.1 |
| 1.3 | Go 端 KeyPayload 解析更新 | P0 | 1h | 1.1 |
| 2.1 | 新增 COMMIT_REQUEST/RESULT 命令 | P1 | 2h | - |
| 2.2 | TSF Barrier 管理 | P1 | 4h | 2.1 |
| 2.3 | Go 端 HandleCommitRequest | P1 | 3h | 2.1 |
| 2.4 | 推送通道（单独管道或轮询） | P1 | 6h | 2.3 |
| 3.1 | 普通字符异步发送 | P2 | 3h | 1.x, 2.x |
| 3.2 | 候选词推送 | P2 | 4h | 2.4 |
| 4.1 | 状态广播机制 | P2 | 3h | 2.4 |
| 4.2 | TSF 处理状态推送 | P2 | 2h | 4.1 |

---

## 兼容性与回退

### 协议版本升级

```cpp
// BinaryProtocol.h
constexpr uint16_t PROTOCOL_VERSION_V1   = 0x1000; // Current
constexpr uint16_t PROTOCOL_VERSION_V1_1 = 0x1001; // With barrier support
```

### 回退策略

1. **Go 服务不可用**：TSF 使用断路器机制，降级为直通模式
2. **Barrier 超时**：500ms 无响应，fallback 到本地提交（可能有副作用）
3. **推送通道不可用**：回退到同步模式

---

## 测试场景

1. **快速输入**："d空格d" 连续输入，验证不会出现字符泄露或候选框残留
2. **模式切换**：Shift 切换中英文，验证状态同步正确
3. **多窗口**：在 Notepad 和 VSCode 之间切换，验证状态一致
4. **IPC 延迟模拟**：人为增加 Go 处理延迟，验证 barrier 机制正常
5. **断开重连**：Go 服务重启，验证 TSF 能正常重连并同步状态

---

## 文件修改清单

### C++ 端

| 文件 | 修改内容 |
|------|----------|
| `wind_tsf/include/BinaryProtocol.h` | 扩展 KeyPayload，新增命令常量 |
| `wind_tsf/include/KeyEventSink.h` | 添加状态机成员，Barrier 管理 |
| `wind_tsf/src/KeyEventSink.cpp` | 状态机实现，Barrier 流程 |
| `wind_tsf/include/IPCClient.h` | 新增异步发送方法，推送接收 |
| `wind_tsf/src/IPCClient.cpp` | 实现异步发送，推送通道 |
| `wind_tsf/src/TextService.cpp` | 处理状态推送 |

### Go 端

| 文件 | 修改内容 |
|------|----------|
| `wind_input/internal/ipc/binary_protocol.go` | 新增命令常量，更新 KeyPayload |
| `wind_input/internal/ipc/binary_codec.go` | 新增编解码方法 |
| `wind_input/internal/bridge/protocol.go` | 新增 CommitRequest/Result 类型 |
| `wind_input/internal/bridge/server.go` | 推送机制，处理 COMMIT_REQUEST |
| `wind_input/internal/coordinator/coordinator.go` | HandleCommitRequest，状态广播 |
