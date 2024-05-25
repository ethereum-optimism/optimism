# User Facing Monitoring - Metamask Tests

## Running Locally

### Building Docker Image

```bash
docker build -t ufm-test-service-metamask .
```

### Running the Docker Container on MacOS

The following steps were taken from [here](https://www.oddbird.net/2022/11/30/headed-playwright-in-docker/#macos)

Apple’s operating system doesn’t include a built-in XServer, but we can use [XQuartz](https://www.xquartz.org/) to provide one:

1. Install XQuartz: `brew install --cask xquartz``
2. Open XQuartz, go to `Preferences -> Security`, and check `Allow connections from network clients`
3. Restart your computer (restarting XQuartz might not be enough)
4. Start XQuartz by executing `xhost +localhost` in your terminal
5. Open Docker Desktop and edit settings to give access to `/tmp/.X11-unix` in `Preferences -> Resources -> File sharing`

Once XQuartz is running with the right permissions, you can populate the environment variable and socket Docker args (these envs are defaulted to the below values in `ufm-test-services/.env.example`):

```bash
docker run --rm -it \
-e DISPLAY=host.docker.internal:0 \
-v /tmp/.X11-unix:/tmp/.X11-unix \
ufm-test-service-metamask
```
