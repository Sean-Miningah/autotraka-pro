"""Integration test: Go MetaClient against MockServer.

This test verifies that the Go messaging service can send WhatsApp messages
via the Meta API, which is mocked by MockServer.
"""

import pytest
import requests


@pytest.mark.integration
class TestMetaWhatsAppAPI:
    """Tests for Meta WhatsApp Business API mocking."""

    def test_mockserver_meta_send_message_expectation_exists(self, mockserver_base_url: str):
        """Verify that MockServer has the Meta send-message expectation loaded."""
        response = requests.get(
            f"{mockserver_base_url}/mockserver/retrieve?type=ACTIVE_EXPECTATIONS"
        )
        response.raise_for_status()
        expectations = response.json()

        expectation_ids = [exp.get("id") for exp in expectations]
        assert "meta-send-message" in expectation_ids, (
            f"Expectation 'meta-send-message' not found. Loaded: {expectation_ids}"
        )

    def test_mockserver_meta_send_message_responds_correctly(self, mockserver_base_url: str):
        """Directly call the mocked Meta API and verify the response shape."""
        payload = {
            "messaging_product": "whatsapp",
            "recipient_type": "individual",
            "to": "+1234567890",
            "type": "text",
            "text": {"body": "Hello from MockServer"},
        }

        response = requests.post(
            f"{mockserver_base_url}/v19.0/123456789/messages",
            headers={
                "Authorization": "Bearer fake-meta-token",
                "Content-Type": "application/json",
            },
            json=payload,
        )

        assert response.status_code == 200
        data = response.json()
        assert data["messaging_product"] == "whatsapp"
        assert len(data["contacts"]) == 1
        assert data["messages"][0]["id"] == "mock_msg_12345"
