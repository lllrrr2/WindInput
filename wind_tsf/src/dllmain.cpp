#include "Globals.h"
#include "ClassFactory.h"
#include "Register.h"
#include "FileLogger.h"

BOOL WINAPI DllMain(HINSTANCE hInstance, DWORD dwReason, LPVOID pvReserved)
{
    switch (dwReason)
    {
        case DLL_PROCESS_ATTACH:
            g_hInstance = hInstance;
            DisableThreadLibraryCalls(hInstance);
            CFileLogger::Instance().Init();
            break;

        case DLL_PROCESS_DETACH:
            CFileLogger::Instance().Shutdown();
            break;
    }

    return TRUE;
}

// DLL 导出函数
STDAPI DllCanUnloadNow()
{
    return (g_lServerLock == 0) ? S_OK : S_FALSE;
}

STDAPI DllGetClassObject(REFCLSID rclsid, REFIID riid, LPVOID* ppv)
{
    if (ppv == nullptr)
        return E_INVALIDARG;

    *ppv = nullptr;

    if (!IsEqualCLSID(rclsid, c_clsidTextService))
        return CLASS_E_CLASSNOTAVAILABLE;

    CClassFactory* pClassFactory = new CClassFactory();
    if (pClassFactory == nullptr)
        return E_OUTOFMEMORY;

    HRESULT hr = pClassFactory->QueryInterface(riid, ppv);
    pClassFactory->Release();

    return hr;
}

STDAPI DllRegisterServer()
{
    return RegisterServer();
}

STDAPI DllUnregisterServer()
{
    return UnregisterServer();
}
