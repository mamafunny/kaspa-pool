docker build -t ${GCP_PROJ}/kaspapool/worker . --build-arg target=poolworker
docker push ${GCP_PROJ}/kaspapool/worker