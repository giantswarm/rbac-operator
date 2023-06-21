import time
import logging


def retry(max_retries=5, delay=10):
    LOGGER = logging.getLogger(__name__)

    if max_retries < 1:
        raise Exception("max_retries must be greater than 0")

    def decorator(func):
        def wrapper(*args, **kwargs):
            retries = 0
            last_exception = Exception("No exception")
            while retries < max_retries:
                try:
                    return func(*args, **kwargs)
                except Exception as e:
                    retries += 1
                    time.sleep(delay)
                    LOGGER.debug(f"Retrying {func.__name__} due to error: {e}")
                    last_exception = e
            raise last_exception

        return wrapper

    return decorator
