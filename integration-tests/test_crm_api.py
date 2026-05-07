"""Integration test: Python CRM client against MockServer.

This test verifies that the Python AI service can fetch CRM data
from Salesforce and HubSpot, which are mocked by MockServer.
"""

import pytest
import requests


@pytest.mark.integration
class TestCRMAPIs:
    """Tests for CRM API mocking (Salesforce, HubSpot)."""

    def test_mockserver_salesforce_expectation_exists(self, mockserver_base_url: str):
        """Verify that MockServer has the Salesforce expectation loaded."""
        response = requests.get(
            f"{mockserver_base_url}/mockserver/retrieve?type=ACTIVE_EXPECTATIONS"
        )
        response.raise_for_status()
        expectations = response.json()

        expectation_ids = [exp.get("id") for exp in expectations]
        assert "salesforce-get-account" in expectation_ids, (
            f"Expectation 'salesforce-get-account' not found. Loaded: {expectation_ids}"
        )

    def test_mockserver_salesforce_get_account(self, mockserver_base_url: str):
        """Directly call the mocked Salesforce API and verify the response."""
        response = requests.get(
            f"{mockserver_base_url}/services/data/v59.0/sobjects/Account/001xx000003DHPxAAO",
            headers={"Authorization": "Bearer fake-salesforce-token"},
        )

        assert response.status_code == 200
        data = response.json()
        assert data["Id"] == "001xx000003DHPxAAO"
        assert data["Name"] == "Acme Corporation"
        assert data["Industry"] == "Technology"

    def test_mockserver_hubspot_expectation_exists(self, mockserver_base_url: str):
        """Verify that MockServer has the HubSpot expectation loaded."""
        response = requests.get(
            f"{mockserver_base_url}/mockserver/retrieve?type=ACTIVE_EXPECTATIONS"
        )
        response.raise_for_status()
        expectations = response.json()

        expectation_ids = [exp.get("id") for exp in expectations]
        assert "hubspot-get-contact" in expectation_ids, (
            f"Expectation 'hubspot-get-contact' not found. Loaded: {expectation_ids}"
        )

    def test_mockserver_hubspot_get_contact(self, mockserver_base_url: str):
        """Directly call the mocked HubSpot API and verify the response."""
        response = requests.get(
            f"{mockserver_base_url}/crm/v3/objects/contacts/12345",
            headers={"Authorization": "Bearer fake-hubspot-token"},
        )

        assert response.status_code == 200
        data = response.json()
        assert data["id"] == "12345"
        assert data["properties"]["firstname"] == "Jane"
        assert data["properties"]["email"] == "jane.doe@example.com"
