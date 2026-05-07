"""Health check endpoints."""

from datetime import datetime, timezone

from fastapi import APIRouter

router = APIRouter()


@router.get("/health")
async def health_check() -> dict:
    return {
        "status": "healthy",
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "version": "0.1.0",
        "checks": {"ai_service": "ok"},
    }


@router.get("/ping")
async def ping() -> dict:
    return {"message": "pong"}
