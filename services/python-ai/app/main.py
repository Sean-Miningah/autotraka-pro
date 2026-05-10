"""FastAPI application factory."""

from fastapi import FastAPI

from app.api import health


def create_app() -> FastAPI:
    app = FastAPI(
        title="Python AI Service",
        description="AI orchestration layer for WhatsApp CRM",
        version="0.1.0",
    )

    app.include_router(health.router, prefix="", tags=["health"])

    return app


app = create_app()
