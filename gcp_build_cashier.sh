docker build -t ${GCP_PROJ}/kaspapool/cashier . --build-arg target=cashier
docker push ${GCP_PROJ}/kaspapool/cashier