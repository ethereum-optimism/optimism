import os

from logging.config import dictConfig

# Retrieve the log level from the environment variable 'LOG_LEVEL'.
# If not set, default to 'INFO'.
log_level = os.getenv('LOG_LEVEL')

# Define the logging configuration as a dictionary.
log_config = {
    'version': 1, # The schema version of the configuration.
    'loggers': {
        '': { # The root logger.
            'handlers': ['console'], # Use the 'console' handler for this logger.
            'level': log_level if log_level is not None else 'INFO', # Set the log level.
        },
    },
    'handlers': {
        'console': { # Define a handler named 'console'.
            'formatter': 'stderr', # Use the 'stderr' formatter for this handler.
            'class': 'logging.StreamHandler', # Use the StreamHandler class.
            'stream': 'ext://sys.stdout' # Output to stdout.
        }
    },
    'formatters': {
        'stderr': { # Define a formatter named 'stderr'.
            'format': '[%(levelname)s|%(asctime)s] %(message)s', # The format of the log message.
            'datefmt': '%m-%d-%Y %I:%M:%S' # The format of the date/time in the log message.
        }
    },
}

# Apply the logging configuration.
dictConfig(log_config)
