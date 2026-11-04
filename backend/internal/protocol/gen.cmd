@echo off
setlocal

set "PROTO_NAMES=sdkws user paper compare"

for %%i in (%PROTO_NAMES%) do (
    protoc --go_out=./%%i --go_opt=module=github.com/GouTouZhao/PaperCompare/internal/protocol/%%i %%i/%%i.proto
    if ERRORLEVEL 1 (
        echo Error processing %%i.proto
        exit /b %ERRORLEVEL%
    )
    protoc --go-grpc_out=./%%i --go-grpc_opt=module=github.com/GouTouZhao/PaperCompare/internal/protocol/%%i %%i/%%i.proto
    if ERRORLEVEL 1 (
        echo Error processing %%i.proto
        exit /b %ERRORLEVEL%
    )
)

for /r %%f in (*.pb.go) do (
    powershell -Command "(Get-Content -Path '%%f' -Encoding UTF8) -replace ',omitempty', '' | Set-Content -Path '%%f' -Encoding UTF8"
)

echo finished all
endlocal
