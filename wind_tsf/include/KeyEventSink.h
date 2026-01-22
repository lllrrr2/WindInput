#pragma once

#include "Globals.h"
#include <string>

class CTextService;

class CKeyEventSink : public ITfKeyEventSink
{
public:
    CKeyEventSink(CTextService* pTextService);
    ~CKeyEventSink();

    // IUnknown
    STDMETHODIMP QueryInterface(REFIID riid, void** ppvObj);
    STDMETHODIMP_(ULONG) AddRef();
    STDMETHODIMP_(ULONG) Release();

    // ITfKeyEventSink
    STDMETHODIMP OnSetFocus(BOOL fForeground);
    STDMETHODIMP OnTestKeyDown(ITfContext* pContext, WPARAM wParam, LPARAM lParam, BOOL* pfEaten);
    STDMETHODIMP OnKeyDown(ITfContext* pContext, WPARAM wParam, LPARAM lParam, BOOL* pfEaten);
    STDMETHODIMP OnTestKeyUp(ITfContext* pContext, WPARAM wParam, LPARAM lParam, BOOL* pfEaten);
    STDMETHODIMP OnKeyUp(ITfContext* pContext, WPARAM wParam, LPARAM lParam, BOOL* pfEaten);
    STDMETHODIMP OnPreservedKey(ITfContext* pContext, REFGUID rguid, BOOL* pfEaten);

    // Initialize/Uninitialize
    BOOL Initialize();
    void Uninitialize();

    // Reset composing state (called when focus is lost or input field changes)
    void ResetComposingState() { _isComposing = FALSE; }

private:
    LONG _refCount;
    CTextService* _pTextService;
    DWORD _dwKeySinkCookie;

    // State
    BOOL _isComposing;
    BOOL _shiftPending;  // True if Shift was pressed alone (for mode toggle on release)

    // Modifier key flags (using KEY_ prefix to avoid Windows macro conflicts)
    static const int KEY_MOD_SHIFT = 0x01;
    static const int KEY_MOD_CTRL  = 0x02;
    static const int KEY_MOD_ALT   = 0x04;

    // Helper methods
    int _GetModifierState();
    BOOL _IsKeyWeShouldHandle(WPARAM wParam);
    BOOL _SendKeyToService(WPARAM wParam);
    void _HandleServiceResponse();
};
