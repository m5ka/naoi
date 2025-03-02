"""Contains settings for the naoi instance, filled via environment variables."""

from environs import Env

env = Env()
env.read_env()

docker_host = env.str("DOCKER_HOST")
cache_path = env.path("CACHE_PATH")
