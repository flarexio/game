# Edge Gaming

## Build Requirements

### Prerequisites

To build this project, you need to install the following tools:

1. **C Compiler**: LLVM-mingw
   - Download from: https://github.com/mstorsjo/llvm-mingw/releases
   - Use the UCRT runtime version

2. **CMake**
   - Download from: https://cmake.org/download/

3. **Ninja**
   - Download from: https://github.com/ninja-build/ninja/releases
   - Or install via MSYS2: `pacman -S ninja`

4. **MSYS2** (for OpenSSL dependency)
   - Download from: https://www.msys2.org/
   - Required because `moonlight-common-c` depends on OpenSSL, which is not included in the minimal LLVM-mingw distribution

### Setup Instructions

1. Install LLVM-mingw (UCRT version) and add it to your PATH
2. Install CMake and add it to your PATH
3. Install Ninja and add it to your PATH
4. Install MSYS2 and install OpenSSL (UCRT version):
   ```bash
   pacman -S --needed mingw-w64-ucrt-x86_64-openssl mingw-w64-ucrt-x86_64-pkg-config
   ```
5. Add MSYS2's UCRT64 bin directory to your PATH (e.g., `C:\msys64\ucrt64\bin`)

### Build

1. Clone the repository and initialize submodules:
   ```bash
   git clone <repository-url>
   cd game
   git submodule update --init --recursive
   ```

2. Build moonlight-common-c (PowerShell):
   ```powershell
   cd thirdparty/moonlight-common-c
   
   # Set environment variables
   $env:CC = "clang"
   $env:CXX = "clang++"
   $env:OPENSSL_ROOT_DIR = "C:/msys64/ucrt64"
   $env:CMAKE_PREFIX_PATH = "C:/msys64/ucrt64"
   $env:PATH = "C:\msys64\ucrt64\bin;$env:PATH"
   
   # Configure and build
   mkdir build; cd build
   cmake -G "Ninja" -S .. -B . `
     -DCMAKE_BUILD_TYPE=Release `
     -DBUILD_SHARED_LIBS=ON `
     -DCMAKE_C_COMPILER=clang `
     -DCMAKE_RC_COMPILER=llvm-rc `
     -DCMAKE_C_COMPILER_TARGET=x86_64-w64-mingw32 `
     -DOPENSSL_ROOT_DIR="C:/msys64/ucrt64" `
     -DOPENSSL_USE_STATIC_LIBS=OFF
   
   cmake --build . --parallel
   ```

3. Set up runtime environment:
   
   Before running the application, ensure the following paths are in your PATH environment variable:
   - `thirdparty\moonlight-common-c\build` (for `moonlight-common-c.dll`)
   - `C:\msys64\ucrt64\bin` (for OpenSSL DLLs: `libssl-3-x64.dll`, `libcrypto-3-x64.dll`)

4. Build the project:
   ```bash
   # Build commands here
   ```

## Sample Video

```bash
ffmpeg -re -stream_loop -1 -i input.mp4 \
  -map 0:v -c:v copy -an -f h264 unix:///tmp/stream/video.sock \
  -map 0:a -vn -c:a libopus -ac 2 -page_duration 20000 -f ogg unix:///tmp/stream/audio.sock
```

## Edge Gaming

```bash
# ffmpeg -hide_banner -h encoder=h264_nvenc
ffmpeg -init_hw_device d3d11va -filter_complex "ddagrab=0:offset_x=1280:offset_y=720:video_size=1280x720:framerate=60" -c:v h264_nvenc -preset p1 -tune ull -f h264 tcp://localhost:3000

# ffmpeg -hide_banner -list_devices true -f dshow -i dummy
# ffmpeg -hide_banner -h encoder=libopus
ffmpeg -f dshow -i audio="立體聲混音 (Realtek High Definition Audio)" -ac 2 -c:a libopus -b:a 64k -application lowdelay -page_duration 2000 -f opus tcp://localhost:3002
```
