#!/bin/bash

docker build -t us-central1-docker.pkg.dev/hayakawa-selenium/hayakawa-docker-repo/go-chatbot .

gcloud auth configure-docker us-central1-docker.pkg.dev

docker push us-central1-docker.pkg.dev/hayakawa-selenium/hayakawa-docker-repo/go-chatbot

# Deploy the container with the environment variables
gcloud run deploy line-chatbot \
  --image us-central1-docker.pkg.dev/hayakawa-selenium/hayakawa-docker-repo/go-chatbot:latest \
  --region us-central1 --cpu-throttling --memory 1024Mi \
  --add-cloudsql-instances hayakawa-selenium:us-central1:linbot-user-token

#   --update-env-vars "CHANNEL_SECRET=CHANNEL_SECRET:1,CHANNEL_TOKEN=CHANNEL_TOKEN:1,OAUTH_CREDS=OAUTH_CREDS:1"

# gcloud run deploy line-chatbot \
#  --image us-central1-docker.pkg.dev/hayakawa-selenium/hayakawa-docker-repo/go-chatbot:latest \
#  --region us-central1 --cpu-throttling --memory 1024Mi \
#  --update-env-vars 'CHANNEL_SECRET=$(gcloud secrets versions access latest --secret CHANNEL_SECRET),CHANNEL_TOKEN=$(gcloud secrets versions access latest --secret CHANNEL_TOKEN),OAUTH_CREDS=$(gcloud secrets versions access latest --secret OAUTH_CREDS)'