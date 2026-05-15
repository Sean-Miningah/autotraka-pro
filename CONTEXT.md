# Autotraka Pro — Domain Glossary

Single-context repo for a multi-channel CRM gateway.

## Meta Graph API

The external API surface for WhatsApp Business, Facebook Messenger, and Instagram messaging.

- **Versioning**: The system targets the latest stable Graph API version. As of this writing, that is **v23.0**. All outbound calls, mock server routes, and test fixtures must use `v23.0`.
- **Base URL**: `https://graph.facebook.com` in production; `http://mockserver:1080` (or equivalent mock) in development / CI.

## Channels

Abstraction over messaging platforms.

- **WhatsApp** — uses the WhatsApp Business API (`/{phoneNumberID}/messages`, `/{wabaID}/message_templates`).
- **Facebook Messenger** — uses the Messenger Platform API (`/{pageID}/messages`).
- **Instagram** — uses the Instagram Conversational API (`/{accountID}/messages`).

## Webhook

Inbound payload pushed by Meta to our system. Signed with HMAC-SHA256 (`X-Hub-Signature-256`). Parsed by `channel.WhatsApp`, `channel.Facebook`, or `channel.Instagram` into a `channel.WebhookEvent`.
