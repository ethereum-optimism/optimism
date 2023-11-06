ARG GCP_PROJECT_ID
ARG GCP_ARTIFACT_REPOSITORY
FROM us-docker.pkg.dev/${GCP_PROJECT_ID}/${GCP_ARTIFACT_REPOSITORY}/images/ci-builder:latest AS builder

COPY ./abigen /usr/local/bin/abigen
