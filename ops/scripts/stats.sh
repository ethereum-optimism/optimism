#!/bin/bash

while true; do
  docker stats --no-stream
  free -m
  ps aux --sort=-%mem | head
  sleep 1;
done
