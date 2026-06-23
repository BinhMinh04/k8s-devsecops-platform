# DevSecOps Lab
cd app
docker build -t devsecops-app:dev .
docker run -d -p 8080:8080 --name myapp devsecops-app:dev
curl localhost:8080/health     # → ok
curl localhost:8080/metrics    # → app_up 1

trivy image devsecops-app:dev --severity HIGH,CRITICAL
