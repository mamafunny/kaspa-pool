kubectl create configmap -n pool worker-config --from-file=worker_config.yml
#kubectl apply -f grafana-deployment.yml