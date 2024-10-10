#include <d3d11.h>
#include <dxgi1_2.h>
#include <windows.h>
#include <wrl.h>
#include <iostream>

// 使用 Microsoft 的 WRL 來簡化 COM 智能指針管理
using namespace Microsoft::WRL;

// 初始化 Direct3D 和 DXGI
bool InitD3D11AndDXGI(ComPtr<ID3D11Device>& device, ComPtr<IDXGIOutputDuplication>& deskDupl) {
    // 創建 D3D11 設備
    ComPtr<ID3D11DeviceContext> context;
    if (FAILED(D3D11CreateDevice(nullptr, D3D_DRIVER_TYPE_HARDWARE, nullptr, 0, nullptr, 0, D3D11_SDK_VERSION, &device, nullptr, &context))) {
        std::cerr << "Failed to create D3D11 device" << std::endl;
        return false;
    }

    // 獲取 DXGI 設備
    ComPtr<IDXGIDevice> dxgiDevice;
    if (FAILED(device.As(&dxgiDevice))) {
        std::cerr << "Failed to get IDXGIDevice" << std::endl;
        return false;
    }

    // 獲取 DXGI 適配器
    ComPtr<IDXGIAdapter> dxgiAdapter;
    if (FAILED(dxgiDevice->GetAdapter(&dxgiAdapter))) {
        std::cerr << "Failed to get IDXGIAdapter" << std::endl;
        return false;
    }

    // 獲取 DXGI 輸出 (顯示器)
    ComPtr<IDXGIOutput> dxgiOutput;
    if (FAILED(dxgiAdapter->EnumOutputs(0, &dxgiOutput))) {
        std::cerr << "Failed to get IDXGIOutput" << std::endl;
        return false;
    }

    // 取得 DXGIOutput1 介面來進行螢幕抓取
    ComPtr<IDXGIOutput1> dxgiOutput1;
    if (FAILED(dxgiOutput.As(&dxgiOutput1))) {
        std::cerr << "Failed to get IDXGIOutput1" << std::endl;
        return false;
    }

    // 複製輸出以進行螢幕擷取
    if (FAILED(dxgiOutput1->DuplicateOutput(device.Get(), &deskDupl))) {
        std::cerr << "Failed to duplicate output" << std::endl;
        return false;
    }

    return true;
}

// 擷取螢幕幀
bool CaptureFrame(ComPtr<IDXGIOutputDuplication>& deskDupl, ComPtr<ID3D11Texture2D>& frameTexture) {
    DXGI_OUTDUPL_FRAME_INFO frameInfo;
    ComPtr<IDXGIResource> desktopResource;

    if (FAILED(deskDupl->AcquireNextFrame(500, &frameInfo, &desktopResource))) {
        std::cerr << "Failed to acquire next frame" << std::endl;
        return false;
    }

    if (FAILED(desktopResource.As(&frameTexture))) {
        std::cerr << "Failed to get frame texture" << std::endl;
        return false;
    }

    deskDupl->ReleaseFrame();

    return true;
}

int main() {
    ComPtr<ID3D11Device> device;
    ComPtr<IDXGIOutputDuplication> deskDupl;

    // 初始化 D3D11 和 DXGI 進行螢幕抓取
    if (!InitD3D11AndDXGI(device, deskDupl)) {
        return -1;
    }

    DXGI_OUTDUPL_DESC desc;
    deskDupl->GetDesc(&desc);

    std::cout << desc.ModeDesc.Width << "x";
    std::cout << desc.ModeDesc.Height << "@";
    std::cout << desc.ModeDesc.RefreshRate.Numerator / desc.ModeDesc.RefreshRate.Denominator << "Hz" << std::endl;

    while (true) {
        ComPtr<ID3D11Texture2D> frameTexture;
        if (CaptureFrame(deskDupl, frameTexture)) {
            std::cout << "capture frame success" << std::endl;
        } else {
            std::cerr << "Failed to capture frame" << std::endl;
        }

        Sleep(1000 / 30);
    }

    return 0;
}
