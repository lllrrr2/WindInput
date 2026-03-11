#ifndef NOMINMAX
#define NOMINMAX
#endif
#include <windows.h>
#include <d2d1.h>
#include <d2d1helper.h>
#include <dwrite.h>
#include <wrl/client.h>

#include <algorithm>
#include <cmath>
#include <cstring>
#include <cstdint>
#include <cwchar>
#include <mutex>
#include <string>
#include <unordered_map>
#include <vector>

#pragma comment(lib, "d2d1.lib")
#pragma comment(lib, "dwrite.lib")

using Microsoft::WRL::ComPtr;

namespace {

constexpr wchar_t kDefaultFontName[] = L"Microsoft YaHei";
constexpr wchar_t kSymbolFontName[] = L"Segoe UI Symbol";

struct FormatKey {
    int size = 0;
    bool symbol = false;

    bool operator==(const FormatKey& other) const {
        return size == other.size && symbol == other.symbol;
    }
};

struct FormatKeyHash {
    size_t operator()(const FormatKey& key) const {
        return (static_cast<size_t>(key.size) << 1) ^ static_cast<size_t>(key.symbol);
    }
};

class Renderer {
public:
    Renderer() {
        valid_ = Initialize();
    }

    bool IsValid() const {
        return valid_;
    }

    void SetFont(const wchar_t* fontName) {
        std::lock_guard<std::mutex> lock(mu_);
        fontName_ = (fontName && fontName[0] != L'\0') ? fontName : kDefaultFontName;
        ClearFormatsLocked();
    }

    void SetFontParams(int weight, float scale) {
        std::lock_guard<std::mutex> lock(mu_);
        fontWeight_ = weight > 0 ? weight : DWRITE_FONT_WEIGHT_NORMAL;
        fontScale_ = scale > 0.0f ? scale : 1.0f;
        ClearFormatsLocked();
    }

    bool MeasureString(const wchar_t* text, int fontSize, bool useSymbol, int* width) {
        if (!text || !width || fontSize <= 0) {
            return false;
        }
        std::lock_guard<std::mutex> lock(mu_);
        if (!valid_) {
            return false;
        }

        ComPtr<IDWriteTextFormat> format = GetTextFormatLocked(fontSize, useSymbol);
        if (!format) {
            return false;
        }

        ComPtr<IDWriteTextLayout> layout;
        HRESULT hr = dwriteFactory_->CreateTextLayout(
            text,
            static_cast<UINT32>(wcslen(text)),
            format.Get(),
            10000.0f,
            1000.0f,
            &layout
        );
        if (FAILED(hr) || !layout) {
            return false;
        }

        DWRITE_TEXT_METRICS metrics{};
        hr = layout->GetMetrics(&metrics);
        if (FAILED(hr)) {
            return false;
        }

        *width = static_cast<int>(std::lround(metrics.widthIncludingTrailingWhitespace));
        return true;
    }

    bool BeginDraw(uint8_t* rgba, int width, int height, int stride) {
        std::lock_guard<std::mutex> lock(mu_);
        if (!valid_ || !rgba || width <= 0 || height <= 0 || stride <= 0) {
            return false;
        }

        EndDrawLocked();

        HDC screenDC = GetDC(nullptr);
        if (!screenDC) {
            return false;
        }

        HDC memDC = CreateCompatibleDC(screenDC);
        ReleaseDC(nullptr, screenDC);
        if (!memDC) {
            return false;
        }

        BITMAPINFO bi{};
        bi.bmiHeader.biSize = sizeof(BITMAPINFOHEADER);
        bi.bmiHeader.biWidth = width;
        bi.bmiHeader.biHeight = -height;
        bi.bmiHeader.biPlanes = 1;
        bi.bmiHeader.biBitCount = 32;
        bi.bmiHeader.biCompression = BI_RGB;

        void* bits = nullptr;
        HBITMAP bitmap = CreateDIBSection(memDC, &bi, DIB_RGB_COLORS, &bits, nullptr, 0);
        if (!bitmap || !bits) {
            DeleteDC(memDC);
            return false;
        }

        HGDIOBJ oldBmp = SelectObject(memDC, bitmap);
        if (!oldBmp) {
            DeleteObject(bitmap);
            DeleteDC(memDC);
            return false;
        }

        auto* dst = static_cast<uint8_t*>(bits);
        for (int y = 0; y < height; ++y) {
            uint8_t* srcRow = rgba + y * stride;
            uint8_t* dstRow = dst + y * width * 4;
            for (int x = 0; x < width; ++x) {
                const int si = x * 4;
                dstRow[si + 0] = srcRow[si + 2];
                dstRow[si + 1] = srcRow[si + 1];
                dstRow[si + 2] = srcRow[si + 0];
                dstRow[si + 3] = 255;
            }
        }

        RECT rect{0, 0, width, height};
        HRESULT hr = renderTarget_->BindDC(memDC, &rect);
        if (FAILED(hr)) {
            SelectObject(memDC, oldBmp);
            DeleteObject(bitmap);
            DeleteDC(memDC);
            return false;
        }

        renderTarget_->SetTransform(D2D1::Matrix3x2F::Identity());
        renderTarget_->BeginDraw();

        drawRGBA_ = rgba;
        drawStride_ = stride;
        drawWidth_ = width;
        drawHeight_ = height;
        drawDC_ = memDC;
        drawBitmap_ = bitmap;
        drawOldBitmap_ = oldBmp;
        drawBits_ = dst;
        inDraw_ = true;
        return true;
    }

    bool DrawString(const wchar_t* text, int x, int y, int fontSize, uint32_t rgba, bool useSymbol) {
        std::lock_guard<std::mutex> lock(mu_);
        if (!valid_ || !inDraw_ || !text || fontSize <= 0) {
            return false;
        }

        ComPtr<IDWriteTextFormat> format = GetTextFormatLocked(fontSize, useSymbol);
        if (!format) {
            return false;
        }

        ComPtr<IDWriteTextLayout> layout;
        HRESULT hr = dwriteFactory_->CreateTextLayout(
            text,
            static_cast<UINT32>(wcslen(text)),
            format.Get(),
            10000.0f,
            1000.0f,
            &layout
        );
        if (FAILED(hr) || !layout) {
            return false;
        }

        DWRITE_TEXT_METRICS metrics{};
        layout->GetMetrics(&metrics);

        UINT32 lineCount = 0;
        float baseline = metrics.height * 0.8f;
        hr = layout->GetLineMetrics(nullptr, 0, &lineCount);
        if ((hr == HRESULT_FROM_WIN32(ERROR_INSUFFICIENT_BUFFER) || hr == E_NOT_SUFFICIENT_BUFFER) && lineCount > 0) {
            std::vector<DWRITE_LINE_METRICS> lines(lineCount);
            if (SUCCEEDED(layout->GetLineMetrics(lines.data(), lineCount, &lineCount)) && !lines.empty()) {
                baseline = lines[0].baseline;
            }
        }

        D2D1_COLOR_F color{};
        color.r = static_cast<float>(rgba & 0xFF) / 255.0f;
        color.g = static_cast<float>((rgba >> 8) & 0xFF) / 255.0f;
        color.b = static_cast<float>((rgba >> 16) & 0xFF) / 255.0f;
        color.a = static_cast<float>((rgba >> 24) & 0xFF) / 255.0f;
        brush_->SetColor(color);

        D2D1_POINT_2F origin{
            static_cast<float>(x),
            static_cast<float>(y) - baseline,
        };
        renderTarget_->DrawTextLayout(origin, layout.Get(), brush_.Get(), D2D1_DRAW_TEXT_OPTIONS_NONE);
        return true;
    }

    bool EndDraw() {
        std::lock_guard<std::mutex> lock(mu_);
        return EndDrawLocked();
    }

    ~Renderer() {
        EndDrawLocked();
    }

private:
    bool Initialize() {
        HRESULT hr = DWriteCreateFactory(
            DWRITE_FACTORY_TYPE_SHARED,
            __uuidof(IDWriteFactory),
            reinterpret_cast<IUnknown**>(dwriteFactory_.GetAddressOf())
        );
        if (FAILED(hr)) {
            return false;
        }

        hr = D2D1CreateFactory(D2D1_FACTORY_TYPE_SINGLE_THREADED, d2dFactory_.GetAddressOf());
        if (FAILED(hr)) {
            return false;
        }

        D2D1_RENDER_TARGET_PROPERTIES props = D2D1::RenderTargetProperties(
            D2D1_RENDER_TARGET_TYPE_DEFAULT,
            D2D1::PixelFormat(DXGI_FORMAT_B8G8R8A8_UNORM, D2D1_ALPHA_MODE_IGNORE)
        );

        hr = d2dFactory_->CreateDCRenderTarget(&props, &renderTarget_);
        if (FAILED(hr)) {
            return false;
        }

        renderTarget_->SetTextAntialiasMode(D2D1_TEXT_ANTIALIAS_MODE_CLEARTYPE);

        hr = renderTarget_->CreateSolidColorBrush(
            D2D1::ColorF(D2D1::ColorF::Black),
            &brush_
        );
        return SUCCEEDED(hr);
    }

    ComPtr<IDWriteTextFormat> GetTextFormatLocked(int fontSize, bool useSymbol) {
        const int scaledSize = (std::max)(1, static_cast<int>(std::lround(fontSize * fontScale_)));
        const FormatKey key{scaledSize, useSymbol};
        auto it = formats_.find(key);
        if (it != formats_.end()) {
            return it->second;
        }

        ComPtr<IDWriteTextFormat> format;
        const wchar_t* family = useSymbol ? kSymbolFontName : fontName_.c_str();
        HRESULT hr = dwriteFactory_->CreateTextFormat(
            family,
            nullptr,
            static_cast<DWRITE_FONT_WEIGHT>(fontWeight_),
            DWRITE_FONT_STYLE_NORMAL,
            DWRITE_FONT_STRETCH_NORMAL,
            static_cast<FLOAT>(scaledSize),
            L"zh-CN",
            &format
        );
        if (FAILED(hr) || !format) {
            return nullptr;
        }

        format->SetTextAlignment(DWRITE_TEXT_ALIGNMENT_LEADING);
        format->SetParagraphAlignment(DWRITE_PARAGRAPH_ALIGNMENT_NEAR);
        formats_.emplace(key, format);
        return format;
    }

    void ClearFormatsLocked() {
        formats_.clear();
    }

    bool EndDrawLocked() {
        if (!inDraw_) {
            return true;
        }

        if (renderTarget_) {
            renderTarget_->EndDraw();
        }

        if (drawRGBA_ && drawBits_) {
            for (int y = 0; y < drawHeight_; ++y) {
                uint8_t* dstRow = drawRGBA_ + y * drawStride_;
                uint8_t* srcRow = drawBits_ + y * drawWidth_ * 4;
                for (int x = 0; x < drawWidth_; ++x) {
                    const int i = x * 4;
                    const uint8_t newR = srcRow[i + 2];
                    const uint8_t newG = srcRow[i + 1];
                    const uint8_t newB = srcRow[i + 0];
                    const uint8_t oldR = dstRow[i + 0];
                    const uint8_t oldG = dstRow[i + 1];
                    const uint8_t oldB = dstRow[i + 2];

                    dstRow[i + 0] = newR;
                    dstRow[i + 1] = newG;
                    dstRow[i + 2] = newB;

                    if (newR != oldR || newG != oldG || newB != oldB) {
                        dstRow[i + 3] = 255;
                    }
                }
            }
        }

        if (drawDC_) {
            SelectObject(drawDC_, drawOldBitmap_);
            DeleteObject(drawBitmap_);
            DeleteDC(drawDC_);
        }

        drawRGBA_ = nullptr;
        drawStride_ = 0;
        drawWidth_ = 0;
        drawHeight_ = 0;
        drawDC_ = nullptr;
        drawBitmap_ = nullptr;
        drawOldBitmap_ = nullptr;
        drawBits_ = nullptr;
        inDraw_ = false;
        return true;
    }

    bool valid_ = false;
    std::mutex mu_;
    std::wstring fontName_ = kDefaultFontName;
    int fontWeight_ = DWRITE_FONT_WEIGHT_NORMAL;
    float fontScale_ = 1.0f;

    ComPtr<IDWriteFactory> dwriteFactory_;
    ComPtr<ID2D1Factory> d2dFactory_;
    ComPtr<ID2D1DCRenderTarget> renderTarget_;
    ComPtr<ID2D1SolidColorBrush> brush_;
    std::unordered_map<FormatKey, ComPtr<IDWriteTextFormat>, FormatKeyHash> formats_;

    bool inDraw_ = false;
    uint8_t* drawRGBA_ = nullptr;
    int drawStride_ = 0;
    int drawWidth_ = 0;
    int drawHeight_ = 0;
    HDC drawDC_ = nullptr;
    HBITMAP drawBitmap_ = nullptr;
    HGDIOBJ drawOldBitmap_ = nullptr;
    uint8_t* drawBits_ = nullptr;
};

float ScaleFromBits(uint32_t bits) {
    float scale = 1.0f;
    static_assert(sizeof(scale) == sizeof(bits), "float size mismatch");
    memcpy(&scale, &bits, sizeof(scale));
    if (!(scale > 0.0f)) {
        scale = 1.0f;
    }
    return scale;
}

} // namespace

extern "C" {

__declspec(dllexport) void* WindDWriteCreateRenderer() {
    try {
        auto* renderer = new Renderer();
        if (!renderer->IsValid()) {
            delete renderer;
            return nullptr;
        }
        return renderer;
    } catch (...) {
        return nullptr;
    }
}

__declspec(dllexport) void WindDWriteDestroyRenderer(void* handle) {
    auto* renderer = static_cast<Renderer*>(handle);
    delete renderer;
}

__declspec(dllexport) BOOL WindDWriteSetFont(void* handle, const wchar_t* fontName) {
    auto* renderer = static_cast<Renderer*>(handle);
    if (!renderer) {
        return FALSE;
    }
    renderer->SetFont(fontName);
    return TRUE;
}

__declspec(dllexport) BOOL WindDWriteSetFontParams(void* handle, int weight, uint32_t scaleBits) {
    auto* renderer = static_cast<Renderer*>(handle);
    if (!renderer) {
        return FALSE;
    }
    renderer->SetFontParams(weight, ScaleFromBits(scaleBits));
    return TRUE;
}

__declspec(dllexport) BOOL WindDWriteMeasureString(
    void* handle,
    const wchar_t* text,
    int fontSize,
    BOOL useSymbol,
    int* width
) {
    auto* renderer = static_cast<Renderer*>(handle);
    if (!renderer) {
        return FALSE;
    }
    return renderer->MeasureString(text, fontSize, useSymbol != FALSE, width) ? TRUE : FALSE;
}

__declspec(dllexport) BOOL WindDWriteBeginDraw(
    void* handle,
    uint8_t* rgba,
    int width,
    int height,
    int stride
) {
    auto* renderer = static_cast<Renderer*>(handle);
    if (!renderer) {
        return FALSE;
    }
    return renderer->BeginDraw(rgba, width, height, stride) ? TRUE : FALSE;
}

__declspec(dllexport) BOOL WindDWriteDrawString(
    void* handle,
    const wchar_t* text,
    int x,
    int y,
    int fontSize,
    uint32_t rgba,
    BOOL useSymbol
) {
    auto* renderer = static_cast<Renderer*>(handle);
    if (!renderer) {
        return FALSE;
    }
    return renderer->DrawString(text, x, y, fontSize, rgba, useSymbol != FALSE) ? TRUE : FALSE;
}

__declspec(dllexport) BOOL WindDWriteEndDraw(void* handle) {
    auto* renderer = static_cast<Renderer*>(handle);
    if (!renderer) {
        return FALSE;
    }
    return renderer->EndDraw() ? TRUE : FALSE;
}

} // extern "C"
