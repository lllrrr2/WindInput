#include "KeyEventSink.h"
#include "TextService.h"
#include "IPCClient.h"
#include <cctype>

CKeyEventSink::CKeyEventSink(CTextService* pTextService)
    : _refCount(1)
    , _pTextService(pTextService)
    , _dwKeySinkCookie(TF_INVALID_COOKIE)
    , _isComposing(FALSE)
    , _shiftPending(FALSE)
{
    _pTextService->AddRef();
}

CKeyEventSink::~CKeyEventSink()
{
    SafeRelease(_pTextService);
}

STDAPI CKeyEventSink::QueryInterface(REFIID riid, void** ppvObj)
{
    if (ppvObj == nullptr)
        return E_INVALIDARG;

    *ppvObj = nullptr;

    if (IsEqualIID(riid, IID_IUnknown) || IsEqualIID(riid, IID_ITfKeyEventSink))
    {
        *ppvObj = (ITfKeyEventSink*)this;
    }

    if (*ppvObj)
    {
        AddRef();
        return S_OK;
    }

    return E_NOINTERFACE;
}

STDAPI_(ULONG) CKeyEventSink::AddRef()
{
    return InterlockedIncrement(&_refCount);
}

STDAPI_(ULONG) CKeyEventSink::Release()
{
    LONG cr = InterlockedDecrement(&_refCount);

    if (cr == 0)
    {
        delete this;
    }

    return cr;
}

STDAPI CKeyEventSink::OnSetFocus(BOOL fForeground)
{
    OutputDebugStringW(L"[WindInput] KeyEventSink::OnSetFocus\n");
    return S_OK;
}

STDAPI CKeyEventSink::OnTestKeyDown(ITfContext* pContext, WPARAM wParam, LPARAM lParam, BOOL* pfEaten)
{
    *pfEaten = FALSE;

    // First check if the context is read-only (browser non-editable area)
    // This is critical for allowing browser shortcuts (Space to scroll, etc.)
    if (_IsContextReadOnly(pContext))
    {
        // Read-only context, don't intercept any keys
        return S_OK;
    }

    // Normal key handling
    *pfEaten = _IsKeyWeShouldHandle(wParam);
    return S_OK;
}

STDAPI CKeyEventSink::OnKeyDown(ITfContext* pContext, WPARAM wParam, LPARAM lParam, BOOL* pfEaten)
{
    *pfEaten = FALSE;

    // Special handling for Shift key (mode toggle on release)
    // Note: We still handle Shift even in read-only context for mode switching
    if (wParam == VK_SHIFT)
    {
        // Check if this is a key repeat (bit 30 of lParam)
        if (lParam & 0x40000000)
        {
            // Key repeat, ignore
            *pfEaten = TRUE;
            return S_OK;
        }

        // Check if other modifiers are pressed (Ctrl+Shift, Alt+Shift are system shortcuts)
        if ((GetAsyncKeyState(VK_CONTROL) & 0x8000) || (GetAsyncKeyState(VK_MENU) & 0x8000))
        {
            _shiftPending = FALSE;
            return S_OK;  // Let system handle it
        }

        // Mark shift as pending (will toggle mode on release if no other key pressed)
        _shiftPending = TRUE;
        *pfEaten = TRUE;
        return S_OK;
    }

    // Any other key cancels shift pending
    _shiftPending = FALSE;

    // Check if context is read-only (browser non-editable area)
    // Don't intercept keys in read-only areas to allow browser shortcuts
    if (_IsContextReadOnly(pContext))
    {
        return S_OK;
    }

    if (!_IsKeyWeShouldHandle(wParam))
    {
        return S_OK;
    }

    // Send key to Go Service
    if (!_SendKeyToService(wParam))
    {
        OutputDebugStringW(L"[WindInput] Failed to send key to service, passing through\n");

        // Service not available - pass through letters directly
        if (wParam >= 'A' && wParam <= 'Z')
        {
            std::wstring ch;
            ch = (wchar_t)towlower((wint_t)wParam);
            _pTextService->InsertText(ch);
            *pfEaten = TRUE;
        }
        return S_OK;
    }

    // Handle response from service
    _HandleServiceResponse();

    *pfEaten = TRUE;
    return S_OK;
}

STDAPI CKeyEventSink::OnTestKeyUp(ITfContext* pContext, WPARAM wParam, LPARAM lParam, BOOL* pfEaten)
{
    // We need to handle Shift key up for mode toggle, and Caps Lock for indicator
    if ((wParam == VK_SHIFT && _shiftPending) || wParam == VK_CAPITAL)
    {
        *pfEaten = TRUE;
    }
    else
    {
        *pfEaten = FALSE;
    }
    return S_OK;
}

STDAPI CKeyEventSink::OnKeyUp(ITfContext* pContext, WPARAM wParam, LPARAM lParam, BOOL* pfEaten)
{
    *pfEaten = FALSE;

    // Handle Shift key release for mode toggle
    if (wParam == VK_SHIFT && _shiftPending)
    {
        _shiftPending = FALSE;

        // Toggle mode
        _pTextService->ToggleInputMode();

        *pfEaten = TRUE;
        return S_OK;
    }

    // Handle Caps Lock key release - show A/a indicator and update language bar
    if (wParam == VK_CAPITAL)
    {
        // Get current Caps Lock state (after the key was processed)
        BOOL capsLockOn = (GetKeyState(VK_CAPITAL) & 0x0001) != 0;

        // Update language bar to show Caps Lock state (A/a in English mode)
        _pTextService->UpdateCapsLockState(capsLockOn);

        // Send to Go service for popup indicator
        CIPCClient* pIPCClient = _pTextService->GetIPCClient();
        if (pIPCClient != nullptr && pIPCClient->IsConnected())
        {
            pIPCClient->SendCapsLockState(capsLockOn);

            // Receive and discard response
            ServiceResponse response;
            pIPCClient->ReceiveResponse(response);
        }

        *pfEaten = TRUE;
        return S_OK;
    }

    return S_OK;
}

STDAPI CKeyEventSink::OnPreservedKey(ITfContext* pContext, REFGUID rguid, BOOL* pfEaten)
{
    *pfEaten = FALSE;
    return S_OK;
}

BOOL CKeyEventSink::Initialize()
{
    OutputDebugStringW(L"[WindInput] KeyEventSink::Initialize\n");

    ITfThreadMgr* pThreadMgr = _pTextService->GetThreadMgr();
    if (pThreadMgr == nullptr)
    {
        OutputDebugStringW(L"[WindInput] ThreadMgr is null!\n");
        return FALSE;
    }

    ITfKeystrokeMgr* pKeystrokeMgr = nullptr;
    HRESULT hr = pThreadMgr->QueryInterface(IID_ITfKeystrokeMgr, (void**)&pKeystrokeMgr);

    if (FAILED(hr) || pKeystrokeMgr == nullptr)
    {
        OutputDebugStringW(L"[WindInput] Failed to get ITfKeystrokeMgr\n");
        return FALSE;
    }

    hr = pKeystrokeMgr->AdviseKeyEventSink(_pTextService->GetClientId(), this, TRUE);
    pKeystrokeMgr->Release();

    if (FAILED(hr))
    {
        OutputDebugStringW(L"[WindInput] AdviseKeyEventSink failed\n");
        return FALSE;
    }

    OutputDebugStringW(L"[WindInput] KeyEventSink initialized successfully\n");
    return TRUE;
}

void CKeyEventSink::Uninitialize()
{
    OutputDebugStringW(L"[WindInput] KeyEventSink::Uninitialize\n");

    ITfThreadMgr* pThreadMgr = _pTextService->GetThreadMgr();
    if (pThreadMgr == nullptr)
        return;

    ITfKeystrokeMgr* pKeystrokeMgr = nullptr;
    if (SUCCEEDED(pThreadMgr->QueryInterface(IID_ITfKeystrokeMgr, (void**)&pKeystrokeMgr)))
    {
        pKeystrokeMgr->UnadviseKeyEventSink(_pTextService->GetClientId());
        pKeystrokeMgr->Release();
    }
}

int CKeyEventSink::_GetModifierState()
{
    int modifiers = 0;

    // Use GetAsyncKeyState to detect modifier key states
    if (GetAsyncKeyState(VK_SHIFT) & 0x8000)
        modifiers |= KEY_MOD_SHIFT;
    if (GetAsyncKeyState(VK_CONTROL) & 0x8000)
        modifiers |= KEY_MOD_CTRL;
    if (GetAsyncKeyState(VK_MENU) & 0x8000)  // VK_MENU = Alt
        modifiers |= KEY_MOD_ALT;

    return modifiers;
}

BOOL CKeyEventSink::_IsKeyWeShouldHandle(WPARAM wParam)
{
    // Get current modifier state
    int modifiers = _GetModifierState();

    // Shift key for Chinese/English toggle
    // Only handle if no other modifier keys are pressed
    if (wParam == VK_SHIFT)
    {
        // If Ctrl or Alt is also pressed, this is a system shortcut (e.g., Shift+Alt)
        // Let the system handle it
        if (modifiers & (KEY_MOD_CTRL | KEY_MOD_ALT))
        {
            OutputDebugStringW(L"[WindInput] Shift with Ctrl/Alt, passing to system\n");
            return FALSE;
        }
        return TRUE;
    }

    // Handle Shift+Space (full-width toggle hotkey)
    if (wParam == VK_SPACE && (modifiers & KEY_MOD_SHIFT) && !(modifiers & (KEY_MOD_CTRL | KEY_MOD_ALT)))
    {
        return TRUE;
    }

    // If Ctrl or Alt is pressed with any key, don't intercept
    // This allows Ctrl+C, Ctrl+V, Alt+Tab, etc. to work
    // Exception: Ctrl+` (VK_OEM_3 = 0xC0) for engine switching
    // Exception: Ctrl+. (VK_OEM_PERIOD = 0xBE) for punctuation toggle
    if (modifiers & (KEY_MOD_CTRL | KEY_MOD_ALT))
    {
        // Allow Ctrl+` for engine switching
        if ((modifiers & KEY_MOD_CTRL) && !(modifiers & KEY_MOD_ALT) && wParam == VK_OEM_3)
        {
            return TRUE;
        }
        // Allow Ctrl+. for punctuation toggle
        if ((modifiers & KEY_MOD_CTRL) && !(modifiers & KEY_MOD_ALT) && wParam == VK_OEM_PERIOD)
        {
            return TRUE;
        }
        return FALSE;
    }

    // Always handle when composing
    if (_isComposing)
    {
        // Handle letters, numbers, backspace, enter, escape, space, and page keys
        if ((wParam >= 'A' && wParam <= 'Z') ||
            (wParam >= '1' && wParam <= '9') ||
            wParam == VK_BACK ||
            wParam == VK_RETURN ||
            wParam == VK_ESCAPE ||
            wParam == VK_SPACE ||
            wParam == VK_OEM_MINUS ||  // - key for page up
            wParam == VK_OEM_PLUS ||   // = key for page down
            _IsPunctuationKey(wParam)) // punctuation keys
        {
            return TRUE;
        }
    }
    else
    {
        // When not composing in Chinese mode, handle letters and punctuation
        if (_pTextService->IsChineseMode())
        {
            if (wParam >= 'A' && wParam <= 'Z')
            {
                return TRUE;
            }
            // Handle punctuation in Chinese mode (for direct punctuation conversion)
            if (_IsPunctuationKey(wParam))
            {
                return TRUE;
            }
        }
        else
        {
            // English mode: only handle letters
            if (wParam >= 'A' && wParam <= 'Z')
            {
                return TRUE;
            }
        }
    }

    return FALSE;
}

// Check if a key is a punctuation key we should handle
BOOL CKeyEventSink::_IsPunctuationKey(WPARAM wParam)
{
    switch (wParam)
    {
    case VK_OEM_COMMA:    // , <
    case VK_OEM_PERIOD:   // . >
    case VK_OEM_1:        // ; :
    case VK_OEM_2:        // / ?
    case VK_OEM_3:        // ` ~
    case VK_OEM_4:        // [ {
    case VK_OEM_5:        // \ |
    case VK_OEM_6:        // ] }
    case VK_OEM_7:        // ' "
        return TRUE;
    default:
        return FALSE;
    }
}

// Convert punctuation virtual key to character
wchar_t CKeyEventSink::_VirtualKeyToPunctuation(WPARAM wParam, BOOL shiftPressed)
{
    switch (wParam)
    {
    case VK_OEM_COMMA:   return shiftPressed ? L'<' : L',';
    case VK_OEM_PERIOD:  return shiftPressed ? L'>' : L'.';
    case VK_OEM_1:       return shiftPressed ? L':' : L';';
    case VK_OEM_2:       return shiftPressed ? L'?' : L'/';
    case VK_OEM_3:       return shiftPressed ? L'~' : L'`';
    case VK_OEM_4:       return shiftPressed ? L'{' : L'[';
    case VK_OEM_5:       return shiftPressed ? L'|' : L'\\';
    case VK_OEM_6:       return shiftPressed ? L'}' : L']';
    case VK_OEM_7:       return shiftPressed ? L'"' : L'\'';
    default:             return 0;
    }
}

BOOL CKeyEventSink::_SendKeyToService(WPARAM wParam)
{
    CIPCClient* pIPCClient = _pTextService->GetIPCClient();
    if (pIPCClient == nullptr)
    {
        OutputDebugStringW(L"[WindInput] IPCClient is null!\n");
        return FALSE;
    }

    // Convert virtual key to character/key name
    std::wstring key;
    int keyCode = (int)wParam;
    BOOL needCaret = FALSE;  // Whether to include caret position in request

    // Get modifier state
    int modifiers = _GetModifierState();
    BOOL shiftPressed = (modifiers & KEY_MOD_SHIFT) != 0;

    if (wParam >= 'A' && wParam <= 'Z')
    {
        // Letter key - determine case based on Caps Lock and Shift state
        BOOL capsLock = (GetKeyState(VK_CAPITAL) & 0x0001) != 0;

        // XOR logic: Caps Lock and Shift cancel each other
        BOOL shouldBeUppercase = capsLock ^ shiftPressed;

        if (_pTextService->IsChineseMode())
        {
            // Chinese mode: always use lowercase for pinyin
            key = (wchar_t)towlower((wint_t)wParam);
            _isComposing = TRUE;
        }
        else
        {
            // English mode: respect Caps Lock and Shift
            if (shouldBeUppercase)
            {
                key = (wchar_t)wParam;  // Already uppercase
            }
            else
            {
                key = (wchar_t)towlower((wint_t)wParam);
            }
        }

        // Always include caret with letter keys (both Chinese and English mode need it)
        // This replaces the separate caret_update call
        needCaret = TRUE;
    }
    else if (wParam >= '1' && wParam <= '9')
    {
        // Number key
        key = (wchar_t)wParam;
    }
    else if (wParam == VK_BACK)
    {
        key = L"backspace";
    }
    else if (wParam == VK_RETURN)
    {
        key = L"enter";
    }
    else if (wParam == VK_ESCAPE)
    {
        key = L"escape";
        _isComposing = FALSE;
    }
    else if (wParam == VK_SPACE)
    {
        key = L"space";
    }
    else if (wParam == VK_SHIFT)
    {
        key = L"shift";
    }
    else if (wParam == VK_OEM_3)  // ` key (backtick/tilde)
    {
        key = L"`";
    }
    else if (wParam == VK_OEM_MINUS)  // - key for page up
    {
        key = L"page_up";
    }
    else if (wParam == VK_OEM_PLUS)  // = key for page down
    {
        key = L"page_down";
    }
    else if (_IsPunctuationKey(wParam))
    {
        // Punctuation key
        wchar_t punct = _VirtualKeyToPunctuation(wParam, shiftPressed);
        if (punct != 0)
        {
            key = punct;
        }
        else
        {
            return FALSE;
        }
    }
    else
    {
        return FALSE;
    }

    // Include caret position in the same request if needed
    if (needCaret)
    {
        LONG x, y, height;
        if (_pTextService->GetCaretPosition(&x, &y, &height))
        {
            return pIPCClient->SendKeyEvent(key, keyCode, modifiers, &x, &y, &height);
        }
    }

    return pIPCClient->SendKeyEvent(key, keyCode, modifiers);
}

void CKeyEventSink::_HandleServiceResponse()
{
    CIPCClient* pIPCClient = _pTextService->GetIPCClient();
    if (pIPCClient == nullptr)
        return;

    ServiceResponse response;
    if (!pIPCClient->ReceiveResponse(response))
    {
        OutputDebugStringW(L"[WindInput] Failed to receive response from service\n");
        return;
    }

    switch (response.type)
    {
    case ResponseType::Ack:
        // ACK is common, no need to log
        break;

    case ResponseType::InsertText:
        {
            _isComposing = FALSE;

            // Insert text into application
            if (!response.text.empty())
            {
                _pTextService->InsertText(response.text);
            }
        }
        break;

    case ResponseType::UpdateComposition:
        OutputDebugStringW(L"[WindInput] Received UpdateComposition from service\n");
        // TODO: Update composition text display
        break;

    case ResponseType::ClearComposition:
        OutputDebugStringW(L"[WindInput] Received ClearComposition from service\n");
        _isComposing = FALSE;
        break;

    case ResponseType::ModeChanged:
        OutputDebugStringW(L"[WindInput] Received ModeChanged from service\n");
        // Clear composing state when mode changes
        _isComposing = FALSE;
        // Update local mode state and language bar icon
        _pTextService->SetInputMode(response.chineseMode);
        break;

    case ResponseType::Consumed:
        // Key was consumed by a hotkey (e.g., Shift+Space, Ctrl+.)
        // Don't output anything, just consume the key
        OutputDebugStringW(L"[WindInput] Key consumed by hotkey\n");
        break;

    default:
        OutputDebugStringW(L"[WindInput] Unknown response type from service\n");
        break;
    }
}

// Check if the current context is read-only (e.g., browser non-editable area)
BOOL CKeyEventSink::_IsContextReadOnly(ITfContext* pContext)
{
    if (!pContext)
    {
        return TRUE;  // No context, treat as read-only for safety
    }

    TF_STATUS tfStatus = {};
    HRESULT hr = pContext->GetStatus(&tfStatus);

    if (SUCCEEDED(hr))
    {
        // Check if the dynamic flags indicate read-only
        if (tfStatus.dwDynamicFlags & TF_SD_READONLY)
        {
            OutputDebugStringW(L"[WindInput] Context is read-only (TF_SD_READONLY)\n");
            return TRUE;
        }

        // Also check for loading state (some apps set this temporarily)
        if (tfStatus.dwDynamicFlags & TF_SD_LOADING)
        {
            OutputDebugStringW(L"[WindInput] Context is loading (TF_SD_LOADING)\n");
            return TRUE;
        }
    }

    return FALSE;  // Context is writable
}

// Check if the current process is a browser (Chrome, Edge, Firefox, etc.)
BOOL CKeyEventSink::_IsCurrentProcessBrowser()
{
    static BOOL s_checked = FALSE;
    static BOOL s_isBrowser = FALSE;

    // Cache the result since process name doesn't change
    if (s_checked)
    {
        return s_isBrowser;
    }

    s_checked = TRUE;
    s_isBrowser = FALSE;

    // Get current process executable name
    wchar_t exePath[MAX_PATH] = {};
    if (GetModuleFileNameW(nullptr, exePath, MAX_PATH) > 0)
    {
        // Extract filename from path
        wchar_t* fileName = wcsrchr(exePath, L'\\');
        if (fileName)
        {
            fileName++;  // Skip the backslash

            // Check against known browser process names (case-insensitive)
            if (_wcsicmp(fileName, L"chrome.exe") == 0 ||
                _wcsicmp(fileName, L"msedge.exe") == 0 ||
                _wcsicmp(fileName, L"firefox.exe") == 0 ||
                _wcsicmp(fileName, L"brave.exe") == 0 ||
                _wcsicmp(fileName, L"opera.exe") == 0 ||
                _wcsicmp(fileName, L"vivaldi.exe") == 0 ||
                _wcsicmp(fileName, L"iexplore.exe") == 0)
            {
                s_isBrowser = TRUE;
                OutputDebugStringW(L"[WindInput] Running in browser process\n");
            }
        }
    }

    return s_isBrowser;
}
