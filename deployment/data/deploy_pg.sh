kubectl create configmap -n data postgres-bootstrap-config --from-file=../../data/pg/pg-init.sql
#kubectl apply -f grafana-deployment.yml