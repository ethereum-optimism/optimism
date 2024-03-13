import os
from logging.config import dictConfig

# Retrieve the log level from an environment variable, default to 'INFO' if not set
log_level = os.getenv('LOG_LEVEL', 'INFO')

# Define the logging configuration as a dictionary
log_config = {
    'version': 1, # Schema version
    'loggers': {
        '': { # Root logger
            'handlers': ['console'], # Handlers to use
            'level': log_level, # Log level, default to 'INFO' if not set
        },
    },
    'handlers': {
        'console': { # Console handler
            'formatter': 'stderr', # Formatter to use
            'class': 'logging.StreamHandler', # Handler class
            'stream': 'ext://sys.stdout', # Output stream
        }
    },
    'formatters': {
        'stderr': { # Formatter for console output
            'format': '[%(levelname)s|%(asctime)s] %(message)s', # Log message format
            'datefmt': '%m-%d-%Y %I:%M:%S', # Date format
        }
    },
}

# Apply the logging configuration
dictConfig(log_config)
