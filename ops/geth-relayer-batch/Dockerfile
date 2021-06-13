FROM isentia/python3.8-aws
RUN apt-get update && apt-get install -y supervisor bash curl jq ca-certificates
RUN curl -sL https://deb.nodesource.com/setup_14.x | bash -
RUN apt-get -y install nodejs
RUN apt-get -y install git binutils
RUN mkdir -p /efs
RUN mkdir -p /root/.ethereum

WORKDIR /efs
RUN git clone https://github.com/aws/efs-utils

WORKDIR /efs/efs-utils
RUN ./build-deb.sh
RUN apt-get -y install ./build/amazon-efs-utils*deb

RUN pip3 install uwsgi
RUN pip3 install flask
RUN pip3 install requests

RUN mkdir /app 
RUN mkdir /app/log
RUN mkdir -p /var/www/flask
RUN mkdir -p /metis

# copy 3 in 1
COPY --from=ethereumoptimism/l2geth:latest /usr/local/bin/geth /usr/local/bin/geth
COPY ./geth.sh /app/geth.sh
RUN chmod +x /app/geth.sh
COPY --from=ethereumoptimism/message-relayer:latest /opt/optimism /opt/optimism
COPY --from=ethereumoptimism/message-relayer:latest /opt/optimism/packages/message-relayer /opt/optimism/packages/message-relayer
COPY ./relayer.sh /app/relayer.sh
RUN chmod +x /app/relayer.sh
COPY --from=ethereumoptimism/batch-submitter:latest /opt/optimism/packages/batch-submitter /opt/optimism/packages/batch-submitter
COPY ./batches.sh /app/batches.sh
RUN chmod +x /app/batches.sh

EXPOSE 8543 8545 8546 8547
COPY src /app
COPY settings/restart.sh /app/restart.sh
RUN chmod +x /app/restart.sh
COPY settings/process_monitor.sh /app/process_monitor.sh
RUN chmod +x /app/process_monitor.sh
COPY settings/process_kill.sh /app/process_kill.sh
RUN chmod +x /app/process_kill.sh
COPY settings/efs-utils.conf /etc/amazon/efs/efs-utils.conf
COPY settings/uwsgi.ini /usr/local/etc/uwsgi/uwsgi.ini
COPY settings/supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY geth.env /app/geth.env
COPY batches.env /app/batches.env

WORKDIR /app
COPY entrypoint.sh /app/entrypoint.sh
COPY clear.sh /app/clear.sh
RUN chmod +x /app/clear.sh
RUN touch crontab.key && echo "*/10 * * * * /app/clear.sh">crontab.key
RUN apt-get update && apt-get install cron -y --no-install-recommends
RUN mkdir -p /app/log
RUN echo "t_geth...">>/app/log/t_geth.log && echo "t_msgrelay...">>/app/log/t_msgrelay.log && echo "t_batch...">>/app/log/t_batch.log && echo "t_supervisord...">>/app/log/t_supervisord.log 
ENTRYPOINT ["/app/entrypoint.sh"]