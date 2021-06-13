#!/bin/bash
while IFS== read -r key value; do
    if [[ -n "$key" ]]; then
        export $key=$value
    fi
done < $1

# start all
t1=$(($(date +%s%N)/1000000))
echo "starting at $t1 .."

echo "/app/geth.sh">>/app/log/t_supervisord.log
/app/geth.sh
t2=$(($(date +%s%N)/1000000))
echo "geth started at $t2 ..">>/app/log/t_supervisord.log
takes=`expr $t2 - $t1`
echo "geth takes $takes ms">>/app/log/t_supervisord.log

echo "/app/relayer.sh">>/app/log/t_supervisord.log
/app/relayer.sh
t2=$(($(date +%s%N)/1000000))
echo "relayer at $t2 ..">>/app/log/t_supervisord.log
takes=`expr $t2 - $t1`
echo "relayer takes $takes ms">>/app/log/t_supervisord.log

echo "/app/batches.sh">>/app/log/t_supervisord.log
/app/batches.sh
t2=$(($(date +%s%N)/1000000))
echo "batches at $t2 ..">>/app/log/t_supervisord.log
takes=`expr $t2 - $t1`
echo "batches takes $takes ms">>/app/log/t_supervisord.log

t2=$(($(date +%s%N)/1000000))
echo "end at $t2 ..">>/app/log/t_supervisord.log
takes=`expr $t2 - $t1`
echo "restart takes $takes ms">>/app/log/t_supervisord.log

nohup /app/process_monitor.sh >>/app/log/t_supervisord.log 2>&1 &
