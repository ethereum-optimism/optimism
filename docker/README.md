# Docker
This directory contains necessary scripts to build and publish each of our docker containers to AWS ECR.

## Publish scripts
The `AWS_ACCOUNT_NUMBER` environment variable will need to be set in order to use these scripts. It's recommended to just add this to your profile.

The only parameter is an optional tag name. If no tag is specified, `latest` will be used.