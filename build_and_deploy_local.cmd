@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion
cd /d "%~dp0"

echo =========================================================
echo [1] 重置 ConfigMap (paper-config)...
echo =========================================================
kubectl.exe delete configmap paper-config 2>nul
kubectl.exe create configmap paper-config --from-file=backend/config
if %errorlevel% neq 0 (
    echo [错误] ConfigMap 创建失败！
    pause
    exit /b 1
)

echo.
echo =========================================================
echo [2] 切换至 Minikube 内部 Docker 引擎构建全套前后端镜像...
echo =========================================================
for /f "tokens=*" %%i in ('minikube -p minikube docker-env --shell cmd') do %%i

cd /d "%~dp0\backend"
docker build -t goutouz/paper-api:latest --build-arg SERVICE=paper-api -f Dockerfile .
docker build -t goutouz/paper-rpc-user:latest --build-arg SERVICE=paper-rpc/paper-rpc-user -f Dockerfile .
docker build -t goutouz/paper-rpc-paper:latest --build-arg SERVICE=paper-rpc/paper-rpc-paper -f Dockerfile .
docker build -t goutouz/paper-rpc-compare:latest --build-arg SERVICE=paper-rpc/paper-rpc-compare -f Dockerfile .

cd /d "%~dp0\frontend"
docker build -t goutouz/paper-frontend:latest -f Dockerfile .

cd /d "%~dp0"
echo.
echo =========================================================
echo [3] 部署全部后端微服务与前端至 Minikube...
echo =========================================================
kubectl.exe delete -f backend/deployments/deploy/ 2>nul
kubectl.exe apply -f backend/deployments/deploy/

echo.
echo [4] 等待所有组件就绪...
kubectl.exe wait --for=condition=available --timeout=120s deployment/paper-api-deployment
kubectl.exe wait --for=condition=available --timeout=120s deployment/paper-frontend-deployment

echo =========================================================
echo 全部前后端 Minikube 部署成功！
echo 可运行:
echo   kubectl port-forward svc/paper-frontend-service 8080:80
echo   kubectl port-forward svc/paper-api-service 10002:10002
echo =========================================================
pause
