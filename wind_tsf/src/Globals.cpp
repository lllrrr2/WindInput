#include "Globals.h"

// 链接必要的库
// 注意：msctf.lib 在新版 SDK 中不需要直接链接
#pragma comment(lib, "ole32.lib")
#pragma comment(lib, "oleaut32.lib")
#pragma comment(lib, "uuid.lib")
#pragma comment(lib, "user32.lib")
#pragma comment(lib, "gdi32.lib")

// 全局变量定义
HINSTANCE g_hInstance = nullptr;
LONG g_lServerLock = 0;

// GUID 定义
// 注意：这些 GUID 应该使用 guidgen.exe 或 uuidgen 生成唯一值
// {7E5A5C60-1234-4567-89AB-CDEF01234567}
const CLSID c_clsidTextService =
    { 0x7e5a5c60, 0x1234, 0x4567, { 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67 } };

// {7E5A5C61-1234-4567-89AB-CDEF01234567}
const GUID c_guidProfile =
    { 0x7e5a5c61, 0x1234, 0x4567, { 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67 } };

// {7E5A5C62-1234-4567-89AB-CDEF01234567}
const GUID c_guidLangBarItemButton =
    { 0x7e5a5c62, 0x1234, 0x4567, { 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67 } };

LONG DllAddRef()
{
    return InterlockedIncrement(&g_lServerLock);
}

LONG DllRelease()
{
    return InterlockedDecrement(&g_lServerLock);
}
