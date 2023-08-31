Steps to Run This:

1. `git clone git@github.com:ethereum-optimism/optimism.git`
2. `git checkout -b wyatt/ufm/init-hello-world origin/wyatt/ufm/init-hello-world`
3. `crontab -e` and copy `optimism/test-services/crontab-example`, replacing `PATH=` with the result of `echo $PATH`
4. Replace `/path/to/docker-compose.yml` with the absolute path to the `optimism/test-services/docker-compose.yml`
    - Should look something like: `/Users/yourUsername/Public/optimism/test-services/docker-compose.yml`
5. Save `crontab` additions
6. Everything should be running automagically in the background 🎉
    - This step may take a while as all the Docker images have to be downloaded and each Test Service container needs to be built
    - You can check if the below UIs are ready by running `docker container ls -a`, you should see something similar to:
```bash
CONTAINER ID   IMAGE                          COMMAND                  CREATED          STATUS                     PORTS                    NAMES
d0c9e4d51ef0   grafana/grafana:latest         "/run.sh"                47 seconds ago   Up 46 seconds              0.0.0.0:3000->3000/tcp   grafana
2e200ea1ee62   prom/prometheus:latest         "/bin/prometheus --c…"   47 seconds ago   Up 46 seconds              0.0.0.0:9090->9090/tcp   prometheus
a1ad8a0c836b   prom/pushgateway               "/bin/pushgateway"       47 seconds ago   Up 46 seconds              0.0.0.0:9091->9091/tcp   test-services-pushgateway-1
```

- You can visit Prometheus UI here: [http://localhost:9090](http://localhost:9090)
- You can visit Grafana UI here: [http://localhost:3000](http://localhost:3000)
    - I don't have a dashboard pre-configured yet, will be implementing this as a part of Milestone no. 2 (#60), but you can verify the data is available by:
        1. Log into Grafana
            - Username: admin
            - Password: adminpassword
        2. Click `Create your first dashboard`, or go here [http://localhost:3000/dashboard/new?orgId=1](http://localhost:3000/dashboard/new?orgId=1)
        3. Click `+ Add visulation`
        4. Click `Prometheus (default)`
        5. On the right you will see `Time series`, click the drop down button (downwards arrow)
        6. Click `Gauge`
        7. Towards the bottome left click the drop down box that says: `Select metric`
        8. Type `go_counter`, and in the top right click `Save` and `Save` again
        - You should see an empty gauge with the number `1`
            - This value was pushed to the Prometheus Push Gateway from the Go container that is running every 5 minutes
