docker build -t ${GCP_PROJ}/kaspapool/worker . --build-arg target=poolworker
docker push ${GCP_PROJ}/kaspapool/worker

docker build -t ${GCP_PROJ}/kaspapool/cashier . --build-arg target=cashier
docker push ${GCP_PROJ}/praxis-paratext-363002/kaspapool/cashier