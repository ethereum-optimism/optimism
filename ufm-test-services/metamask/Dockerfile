# Using the Playwright image
FROM mcr.microsoft.com/playwright:v1.37.1-jammy

# Setting the working directory
WORKDIR /app

# Update PATH
ENV PATH /app/node_modules/.bin:$PATH

RUN npm i -g pnpm

RUN if [ "$METAMASK_PLAYWRIGHT_RUN_HEADLESS" != "false" ]; then \
        apt-get update && \
        apt-get install -y xvfb && \
        rm -rf /var/lib/apt/lists/* ; \
    fi

# Copy necessary files and directories
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml /app/
RUN pnpm install --frozen-lockfile --ignore-scripts
COPY tests /app/tests/
COPY playwright.config.ts /app/
COPY start.sh /app/
COPY tsconfig.json /app/

# Start the script
CMD /bin/bash /app/start.sh
