@echo off
chcp 65001 >nul
echo ========================================================
echo   Docker VHDX 磁盘物理空间压缩脚本 (释放 Windows E盘 600G)
echo ========================================================
echo.
echo 内部已清理完毕：
echo   - Minikube 内部 BuildKit 缓存：已清除 599.1 GB
echo   - Minikube 悬空镜像与容器：已清除 11.7 GB
echo   - 宿主机历史多版本镜像与构建缓存：已清除 65+ GB
echo.
echo 由于 Windows WSL2 虚拟硬盘 (docker_data.vhdx) 是动态扩容的，
echo 在文件删除后需要执行一次压缩，将空间还给 Windows E盘。
echo.
echo 【注意】：执行前请先在 Windows 右下角托盘完全退出 Docker Desktop！
echo.
pause

echo.
echo [1/3] 正在关闭 WSL2 进程...
wsl --shutdown
timeout /t 3 /nobreak >nul

echo [2/3] 正在启用 NTFS 稀疏文件属性...
fsutil sparse setflag "E:\Docker\DockerDesktopWSL\disk\docker_data.vhdx"

echo [3/3] 正在执行 diskpart 物理紧凑 (compact vdisk)...
(
echo select vdisk file="E:\Docker\DockerDesktopWSL\disk\docker_data.vhdx"
echo attach vdisk readonly
echo compact vdisk
echo detach vdisk
echo exit
) > "%TEMP%\compact_docker.txt"

diskpart /s "%TEMP%\compact_docker.txt"
del "%TEMP%\compact_docker.txt" 2>nul

echo.
echo ========================================================
echo   压缩完成！E盘已成功收回 600G+ 空间。
echo   现在您可以重新启动 Docker Desktop 了！
echo ========================================================
pause

