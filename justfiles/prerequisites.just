import 'default.just'

MIN_DOCKER_VERSION := "28.0.0"

# Check if mise is installed and if not, install it
_check_mise:
    #!/usr/bin/env bash
    set -e
    
    # We have multiple levels of autofix (true and nuke)
    if [ -n "$AUTOFIX" ] && [ "$AUTOFIX" != "false" ]; then
        RUN_AUTOFIX="true"
    fi
    
    if ! command -v mise &> /dev/null; then
        echo "Warning: mise is not installed or not in PATH"
        if [ "$RUN_AUTOFIX" = "true" ]; then
            echo "Automatically installing mise..."
            if [ "{{os()}}" = "macos" ] && command -v brew &> /dev/null; then
                echo "Installing mise via Homebrew..."
                brew install mise
            elif [ "{{os_family()}}" = "unix" ]; then
                if command -v curl &> /dev/null; then
                    echo "Installing mise via curl..."
                    curl https://mise.run | sh
                else
                    echo "Error: curl not found. Please install curl first."
                    exit 1
                fi
            else
                echo "Error: Unsupported operating system"
                echo "Please visit https://mise.jdx.dev/getting-started.html for installation instructions"
                exit 1
            fi
        else
            echo "Please install mise and ensure it's properly configured"
            echo "Visit https://mise.jdx.dev/getting-started.html for installation instructions"
            exit 0
        fi
    fi

    if [ -z "$MISE_SHELL" ]; then
        echo "Warning: Not running in a mise-configured shell"
        if [ "$RUN_AUTOFIX" = "true" ]; then
            # We're running in a shebang subshell, so we can't activate mise
            exit 0
        else
            echo "Please run 'mise activate' first to activate the mise environment"
            exit 0
        fi
    fi

    if [ "$RUN_AUTOFIX" = "true" ]; then
        echo "Automatically running mise doctor..."
        # This will exit if mise is outdated, so better run it only if AUTOFIX is true
        mise doctor --silent
    fi

    if mise ls | grep -q "outdated"; then
        echo "Warning: Some tools are not up to date with mise.toml"
        if [ "$RUN_AUTOFIX" = "true" ]; then
            echo "Automatically updating tools..."
            mise install
        else
            echo "Please run 'mise install' to update your tools"
        fi
    fi

# Check if docker is installed and check the version
_check_docker:
    #!/usr/bin/env bash
    if ! command -v docker &> /dev/null; then
        echo "Warning: Docker is not installed"
        if [ "{{os()}}" = "macos" ] && command -v brew &> /dev/null; then
            echo "You can install Docker via Homebrew: brew install docker"
        else
            echo "You can install Docker from https://docs.docker.com/get-docker/"
        fi
        echo "Note: Some features may require Docker to be installed"
        exit 0
    fi
    
    DOCKER_CLIENT_VERSION=$(docker version --format '{{{{.Client.Version}}')
    
    if [ -z "$DOCKER_CLIENT_VERSION" ]; then
        echo "Warning: Could not determine Docker client version"
        echo "Note: Some features may not work correctly"
        exit 0
    fi
    
    if [ "$(printf '%s\n' "{{MIN_DOCKER_VERSION}}" "$DOCKER_CLIENT_VERSION" | sort -V | head -n1)" = "$DOCKER_CLIENT_VERSION" ]; then
        echo "Warning: Docker client version $DOCKER_CLIENT_VERSION is older than recommended version {{MIN_DOCKER_VERSION}}"
        echo "Note: Some features may not work correctly with older Docker versions"
        exit 0
    fi

# Check if mise and docker are installed and if not, install them
_prerequisites: _check_mise _check_docker
    @echo "Environment checks passed"
