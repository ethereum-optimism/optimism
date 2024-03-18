#!/bin/bash

if [ "$METAMASK_PLAYWRIGHT_RUN_HEADLESS" != "false" ]; then
    # Start Xvfb in the background on display :99
    Xvfb :99 &

    # Set the DISPLAY environment variable
    export DISPLAY=:99
fi

npm test

# If something goes wrong, Playwright generates this file, but only if there is an error.
# npx playwright show-trace will log the Playwright error
if [ -f "test-results/metamask-Setup-wallet-and-dApp-chromium-retry1/trace.zip" ]; then
  npx playwright show-trace "test-results/metamask-Setup-wallet-and-dApp-chromium-retry1/trace.zip"
fi

