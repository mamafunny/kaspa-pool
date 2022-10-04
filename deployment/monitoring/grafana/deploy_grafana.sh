kubectl create configmap -n monitoring grafana-ini-config --from-file=grafana.ini
kubectl apply -f grafana-deployment.yml