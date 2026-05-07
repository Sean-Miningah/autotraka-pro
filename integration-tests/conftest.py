"""Integration tests using MockServer for external API mocking.

These tests verify that our services correctly interact with external APIs
(Meta WhatsApp, Salesforce, HubSpot) by mocking them via MockServer.

Run with: pytest integration-tests/ -v
Requires: MockServer running (via docker-compose)
"""

import os

import pytest


@pytest.fixture(scope="session")
def mockserver_base_url() -> str:
    """Return the MockServer base URL."""
    return os.getenv("MOCKSERVER_URL", "http://localhost:1080")
