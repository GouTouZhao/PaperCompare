@echo off
chcp 65001 >nul
cd /d "%~dp0"

echo [1] 重置 ConfigMap (paper-config)...
kubectl.exe delete configmap paper-config 2>nul
kubectl.exe create configmap paper-config --from-file=backend/config

echo [2] 重置所有的 Deployment, Service, Secret, Role...
kubectl.exe delete -f backend/deployments/deploy/ 2>nul
kubectl.exe apply -f backend/deployments/deploy/

echo [3] 等待 paper-api-deployment 与 paper-frontend-deployment 就绪...
kubectl.exe wait --for=condition=available --timeout=120s deployment/paper-api-deployment
kubectl.exe wait --for=condition=available --timeout=120s deployment/paper-frontend-deployment

echo =========================================================
echo [4] 端口映射说明:
echo   kubectl.exe port-forward svc/paper-frontend-service 8080:80
echo   kubectl.exe port-forward svc/paper-api-service 10002:10002
echo =========================================================
pause

