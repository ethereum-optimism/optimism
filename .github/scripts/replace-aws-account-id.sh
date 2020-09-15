#!/bin/bash

sed -iE 's/<AWS_CI_AWS_ACCOUNT_ID>/'"$AWS_ACCOUNT_ID"'/g' ./docker-compose.ci.yml

