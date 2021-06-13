#!/bin/bash
ps -ef | grep process_monitor.sh | grep -v grep | awk '{print $2}'|xargs kill -9
ps -ef | grep geth | grep verbosity | grep -v grep | awk '{print $2}'|xargs kill -9
ps -ef | grep run-batch-submitter.js | grep -v grep | awk '{print $2}'|xargs kill -9
ps -ef | grep run-message-relayer.js | grep -v grep | awk '{print $2}'|xargs kill -9
