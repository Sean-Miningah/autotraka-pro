"""Application configuration via Pydantic Settings."""

from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    port: int = 8081
    env: str = "development"
    database_url: str = "postgresql://devuser:devpass@localhost:5432/wacrm"
    redis_url: str = "redis://localhost:6379/0"
    nats_url: str = "nats://localhost:4222"
    crm_base_url: str = "http://localhost:1080"

    class Config:
        env_prefix = ""
        case_sensitive = False


settings = Settings()
