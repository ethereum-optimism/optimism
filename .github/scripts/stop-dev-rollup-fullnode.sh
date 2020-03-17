#!/bin/bash

TASKS=$(aws ecs list-tasks --cluster dev-rollup-full-node --service-name rollup-full-node)

TASK_ARN=$(echo $TASKS | awk -F\[ '{print $2}' | awk -F\" '{print $2}')

if [ ! -z "${TASK_ARN}" ]; then
  aws ecs stop-task --cluster dev-rollup-full-node --task $TASK_ARN
fi