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

## Desk

The web dashboard that agents use to manage conversations, contacts, and templates. Consumes the go-gateway API.

- **MVP scope**: Auth, Conversations (with real-time via WebSocket), Contacts (list/view), Templates (list/select), Analytics overview.
- **Post-MVP**: Broadcasts, Automations (flow builder), Channel Health, Contact merge.
- **Design language**: Neo-brutalism — bold outlines, saturated accent colors, offset drop-shadows, thick black borders on cards, strong typographic hierarchy. Functional readability for extended agent use takes priority over decorative harshness.
- **Navigation**: Mobile-first — bottom tab nav (Inbox, Contacts, My Stats, Profile) for agents. Desktop — left sidebar nav (Inbox, Contacts, Analytics, Templates, Settings) with 2+1 inbox layout (conversation list + message thread, contact detail as slide-over drawer).
- **Auth flow**: Email → password → tenant picker (if multiple tenants). SvelteKit server proxies auth endpoints and sets httpOnly cookie for refresh token. Client receives a short-lived access token. Requires adding `GET /api/v1/auth/tenants?email=...` to the go-gateway API.
- **Message composer**: Single text input with a "Use Template" button. Template picker opens a searchable dropdown; selecting a template renders parameter fields inline. Free text is the default mode.
- **Conversation status**: Status pill at top of thread (read-only, shows current state) + action buttons near composer for transitions (Resolve, Escalate, Close). Mobile-thumb-reachable.
- **Inbox filtering**: Status tabs (Open / Pending / Escalated / Resolved) at top of conversation list, plus a filter icon for handled_by and assigned member.
- **Contact detail**: Read-only drawer showing contact name, phone, channel badge, tags, custom fields. No editing in MVP.
- **Analytics (admin/desktop)**: Overview metric cards + conversation trend line chart by channel. Date range picker.
- **Analytics (agent/mobile)**: Personal stats only — own open/escalated counts, messages sent today, avg response time. Not team-wide metrics.

## Frontend architecture

- **Framework**: SvelteKit with Svelte 5 (runes mode), TypeScript, Tailwind v4.
- **Data layer**: SvelteKit server (adapter-node) proxies auth — login, register, refresh set httpOnly cookies for the refresh token and return a short-lived access token to the client. Client calls go-gateway directly for all data APIs using the access token. WebSocket auth uses the same access token.
- **Real-time**: Direct WebSocket connection to go-gateway, wrapped in a Svelte store (`src/lib/stores/websocket.ts`). Handles reconnection with exponential backoff, authentication, and event dispatch to relevant stores.
- **Project structure**: Shared UI components in `src/lib/ui/`, API client in `src/lib/api/`, shared stores in `src/lib/stores/`. Features expressed through `src/routes/` not feature directories.
- **Deployment**: Node.js server running SvelteKit (adapter-node). Serves the PWA and handles auth proxying. Containerized alongside go-gateway or deployed separately.

## PWA

- **Installable**: Web app manifest with `display: standalone`, icons, theme color. Agents can "Add to Home Screen" for fullscreen native-like experience.
- **Service worker**: Caches app shell and static assets. Offline shows a "You're offline" page. No offline data access or message queuing in MVP.
- **Push notifications**: Deferred to post-MVP (requires go-gateway web push integration).

## Routes

- `/` — redirects to `/inbox`
- `/inbox` — conversation list (mobile) / list + thread (desktop)
- `/inbox/[id]` — conversation thread + composer (mobile replaces list; desktop shows in center panel)
- `/contacts` — contact list
- `/contacts/[id]` — contact detail read-only drawer
- `/analytics` — overview metrics + trend chart (admin: full team; agent: personal stats, shown as "My Stats" on mobile)
- `/templates` — template list and selector
- `/settings` — profile, theme toggle, logout
- `/auth/login` — email + password
- `/auth/register` — tenant name + email + password
- `/auth/tenants` — tenant picker (if multiple)

## i18n

- **MVP**: English only. Hard-coded strings. Paraglide infrastructure remains in place but unused; translations added post-MVP.

## Visual language

- **Primary accent**: Electric yellow `#FFE600`
- **Secondary accent**: Hot pink `#FF6B9D`
- **Success**: Lime `#BFFF00`
- **Danger**: Red `#FF3333`
- **Light mode**: White base `#FFFFFF`, black text `#000000`, off-white cards `#F5F5F0`
- **Dark mode**: Near-black base `#0A0A0A`, off-white text `#F0F0F0`, deep-gray cards `#1A1A1A`, brightened accent colors for contrast
- **Channel badges**: Official brand colors unchanged — WhatsApp `#25D366`, Instagram `#E4405F`, Facebook `#1877F2`
- **Borders**: 2px solid black (light mode), 2px solid off-white (dark mode)
- **Shadows**: 4px 4px 0px black offset (light mode), 4px 4px 0px off-white (dark mode)
- **Typography**: Space Grotesk for headings and CTAs, system font stack for body. Bold/black weights for hierarchy, regular for body text.

## Conversation

A threaded exchange of messages between a contact and one or more agents on a channel. Has status (open, pending, escalated, resolved, closed) and handled-by (ai, human, hybrid).
_Avoid_: Thread, ticket, chat.

## Contact

A person or organization that communicates via one or more channels. Identified by ChannelIdentities (phone number, page-scoped ID, etc.).
_Avoid_: Customer, client, user (those mean Member, a Tenant's staff).

## Template

A message template synced with Meta. Has status (draft, pending, approved, rejected), language, body, and parameters.
_Avoid_: Saved reply, quick reply.

## Member

A user account within a Tenant who can log in to the Desk and handle conversations.
_Avoid_: Agent, user, staff.

## Tenant

An organization that owns channels, contacts, conversations, and members. Multi-tenancy is enforced at the API level via JWT claims.
_Avoid_: Organization, workspace, account.