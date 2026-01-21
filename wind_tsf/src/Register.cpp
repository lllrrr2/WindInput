#include "Register.h"
#include <shlwapi.h>
#include <strsafe.h>

#pragma comment(lib, "shlwapi.lib")

// COM 服务器注册
static HRESULT RegisterCOMServer()
{
    HRESULT hr = E_FAIL;
    WCHAR szModule[MAX_PATH];
    WCHAR szCLSID[39];

    if (GetModuleFileNameW(g_hInstance, szModule, ARRAYSIZE(szModule)) == 0)
        return E_FAIL;

    // 转换 CLSID 为字符串
    StringFromGUID2(c_clsidTextService, szCLSID, ARRAYSIZE(szCLSID));

    // 注册 CLSID
    WCHAR szKey[256];
    StringCchPrintfW(szKey, ARRAYSIZE(szKey), L"CLSID\\%s", szCLSID);

    HKEY hKey;
    LONG result = RegCreateKeyExW(HKEY_CLASSES_ROOT, szKey, 0, nullptr,
                                   REG_OPTION_NON_VOLATILE, KEY_WRITE, nullptr, &hKey, nullptr);

    if (result == ERROR_SUCCESS)
    {
        RegSetValueExW(hKey, nullptr, 0, REG_SZ, (BYTE*)TEXTSERVICE_DESC,
                       (lstrlenW(TEXTSERVICE_DESC) + 1) * sizeof(WCHAR));
        RegCloseKey(hKey);

        // 注册 InprocServer32
        StringCchPrintfW(szKey, ARRAYSIZE(szKey), L"CLSID\\%s\\InprocServer32", szCLSID);
        result = RegCreateKeyExW(HKEY_CLASSES_ROOT, szKey, 0, nullptr,
                                 REG_OPTION_NON_VOLATILE, KEY_WRITE, nullptr, &hKey, nullptr);

        if (result == ERROR_SUCCESS)
        {
            RegSetValueExW(hKey, nullptr, 0, REG_SZ, (BYTE*)szModule,
                           (lstrlenW(szModule) + 1) * sizeof(WCHAR));
            RegSetValueExW(hKey, L"ThreadingModel", 0, REG_SZ, (BYTE*)L"Apartment",
                           (lstrlenW(L"Apartment") + 1) * sizeof(WCHAR));
            RegCloseKey(hKey);
            hr = S_OK;
        }
    }

    return hr;
}

static HRESULT UnregisterCOMServer()
{
    WCHAR szCLSID[39];
    WCHAR szKey[256];

    StringFromGUID2(c_clsidTextService, szCLSID, ARRAYSIZE(szCLSID));

    StringCchPrintfW(szKey, ARRAYSIZE(szKey), L"CLSID\\%s", szCLSID);
    SHDeleteKeyW(HKEY_CLASSES_ROOT, szKey);

    return S_OK;
}

HRESULT RegisterProfile()
{
    ITfInputProcessorProfiles* pProfiles = nullptr;
    HRESULT hr = CoCreateInstance(CLSID_TF_InputProcessorProfiles, nullptr, CLSCTX_INPROC_SERVER,
                                   IID_ITfInputProcessorProfiles, (void**)&pProfiles);

    if (SUCCEEDED(hr))
    {
        hr = pProfiles->Register(c_clsidTextService);

        if (SUCCEEDED(hr))
        {
            hr = pProfiles->AddLanguageProfile(c_clsidTextService,
                                                TEXTSERVICE_LANGID,
                                                c_guidProfile,
                                                TEXTSERVICE_NAME,
                                                lstrlenW(TEXTSERVICE_NAME),
                                                nullptr, // 图标文件路径
                                                0,       // 图标文件路径长度
                                                0);      // 图标索引
        }

        pProfiles->Release();
    }

    return hr;
}

HRESULT UnregisterProfile()
{
    ITfInputProcessorProfiles* pProfiles = nullptr;
    HRESULT hr = CoCreateInstance(CLSID_TF_InputProcessorProfiles, nullptr, CLSCTX_INPROC_SERVER,
                                   IID_ITfInputProcessorProfiles, (void**)&pProfiles);

    if (SUCCEEDED(hr))
    {
        hr = pProfiles->Unregister(c_clsidTextService);
        pProfiles->Release();
    }

    return hr;
}

HRESULT RegisterCategories()
{
    ITfCategoryMgr* pCategoryMgr = nullptr;
    HRESULT hr = CoCreateInstance(CLSID_TF_CategoryMgr, nullptr, CLSCTX_INPROC_SERVER,
                                   IID_ITfCategoryMgr, (void**)&pCategoryMgr);

    if (SUCCEEDED(hr))
    {
        // 注册为 TIP (Text Input Processor)
        hr = pCategoryMgr->RegisterCategory(c_clsidTextService,
                                             GUID_TFCAT_TIP_KEYBOARD,
                                             c_clsidTextService);

        pCategoryMgr->Release();
    }

    return hr;
}

HRESULT UnregisterCategories()
{
    ITfCategoryMgr* pCategoryMgr = nullptr;
    HRESULT hr = CoCreateInstance(CLSID_TF_CategoryMgr, nullptr, CLSCTX_INPROC_SERVER,
                                   IID_ITfCategoryMgr, (void**)&pCategoryMgr);

    if (SUCCEEDED(hr))
    {
        hr = pCategoryMgr->UnregisterCategory(c_clsidTextService,
                                               GUID_TFCAT_TIP_KEYBOARD,
                                               c_clsidTextService);

        pCategoryMgr->Release();
    }

    return hr;
}

HRESULT RegisterServer()
{
    HRESULT hr;

    hr = CoInitialize(nullptr);
    if (FAILED(hr))
        return hr;

    // 注册 COM 服务器
    hr = RegisterCOMServer();
    if (FAILED(hr))
        goto Exit;

    // 注册配置文件
    hr = RegisterProfile();
    if (FAILED(hr))
        goto Exit;

    // 注册分类
    hr = RegisterCategories();

Exit:
    CoUninitialize();
    return hr;
}

HRESULT UnregisterServer()
{
    HRESULT hr;

    hr = CoInitialize(nullptr);
    if (FAILED(hr))
        return hr;

    // 卸载分类
    UnregisterCategories();

    // 卸载配置文件
    UnregisterProfile();

    // 卸载 COM 服务器
    UnregisterCOMServer();

    CoUninitialize();
    return S_OK;
}
