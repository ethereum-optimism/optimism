#!/bin/bash

while true; do
  docker stats --no-stream
  free -m
  sleep 1;
done
