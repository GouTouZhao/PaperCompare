@echo off
chcp 65001 >nul
echo =========================================================
echo 正在启动 PaperCompare Minikube 本地端口映射:
echo   1. 前端 Web 访问地址: http://localhost:8080
echo   2. 后端 API 网关地址: http://localhost:10002
echo =========================================================

start "PaperCompare-Frontend (8080)" cmd /k "kubectl.exe port-forward svc/paper-frontend-service 8080:80"
start "PaperCompare-Backend (10002)" cmd /k "kubectl.exe port-forward svc/paper-api-service 10002:10002"

echo.
echo 两个端口转发窗口已弹出运行！请勿关闭弹出的窗口。
echo 现在您可以在浏览器中打开: http://localhost:8080
echo =========================================================
pause

