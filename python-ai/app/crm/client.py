"""CRM client stubs."""

import httpx


class CRMClient:
    """Base CRM client for external API interactions."""

    def __init__(self, base_url: str, token: str):
        self.base_url = base_url.rstrip("/")
        self.token = token
        self.client = httpx.AsyncClient()

    async def get_account(self, account_id: str) -> dict:
        """Fetch an account from Salesforce."""
        url = f"{self.base_url}/services/data/v59.0/sobjects/Account/{account_id}"
        headers = {"Authorization": f"Bearer {self.token}"}
        response = await self.client.get(url, headers=headers)
        response.raise_for_status()
        return response.json()

    async def get_contact(self, contact_id: str) -> dict:
        """Fetch a contact from HubSpot."""
        url = f"{self.base_url}/crm/v3/objects/contacts/{contact_id}"
        headers = {"Authorization": f"Bearer {self.token}"}
        response = await self.client.get(url, headers=headers)
        response.raise_for_status()
        return response.json()

    async def close(self):
        await self.client.aclose()
