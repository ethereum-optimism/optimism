#!/bin/bash
echo "source env file"
source /app/geth.env
source /app/batches.env
echo "crontab starting..."
service cron restart
crontab -u root /app/crontab.key
echo "supervisord starting..."
nohup supervisord -c /etc/supervisor/conf.d/supervisord.conf>/app/log/t_supervisord.log &
echo "log tail..."
tail -f /app/log/t_*.log