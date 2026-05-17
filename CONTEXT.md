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
- **Design language**: Minimalist Corporate — clean surfaces, subtle 1px borders, soft elevation shadows, 8px rounded corners, Inter typography, WhatsApp-inspired green palette. High-density information display without cognitive overload. Surfaces are crisp and intentional, using subtle depth to distinguish between navigation, list management, and active conversation threads. Functional readability for extended agent use takes priority over decorative elements.
- **Navigation**: Mobile-first — bottom tab nav (Inbox, Customers, Dashboards, Copilots, Settings) for agents. Desktop — left sidebar nav (Dashboards, Inbox, Customers, Analytics, Copilots, Settings) with 2+1 inbox layout (conversation list + message thread, contact detail as slide-over drawer).
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

- `/` — redirects to `/dashboards`
- `/dashboards` — overview dashboard with key metrics
- `/inbox` — conversation list (mobile) / list + thread (desktop)
- `/inbox/[id]` — conversation thread + composer (mobile replaces list; desktop shows in center panel)
- `/customers` — customer list (was `/contacts`)
- `/customers/[id]` — customer detail read-only drawer
- `/analytics` — overview metrics + trend chart (admin: full team; agent: personal stats, shown as "My Stats" on mobile)
- `/copilots` — AI-powered tools (flow builder, AI assistant — stub, post-MVP)
- `/settings` — profile, logout
- `/auth/login` — email + password
- `/auth/register` — tenant name + email + password
- `/auth/tenants` — tenant picker (if multiple)

## i18n

- **MVP**: English only. Hard-coded strings. Paraglide infrastructure remains in place but unused; translations added post-MVP.

## Visual language

- **Style**: Minimalist Corporate — clean workspace metaphor with tools within reach but out of the way
- **Primary accent**: Deep green `#006d2f`, container green `#25d366`
- **Secondary accent**: Teal `#006b5f`, container teal `#8cf1e1`
- **Tertiary**: Dark teal `#1c695f`, container teal `#7ec5b8`
- **Error**: Material red `#ba1a1a`, container `#ffdad6`
- **Light mode only (MVP)**: Surface `#f8f9fb`, white cards `#ffffff`, near-black text `#191c1e`
- **Channel badges**: Official brand colors unchanged — WhatsApp `#25D366`, Instagram `#E4405F`, Facebook `#1877F2`
- **Borders**: 1px solid `#bbcbb9` (outline-variant)
- **Elevation**: Level 1 `0 1px 3px rgba(0,0,0,0.08)` (cards), Level 2 `0 4px 12px rgba(0,0,0,0.12)` (modals/popovers)
- **Radius**: 8px default, 4px for status pills, 9999px for badge/pill shapes
- **Typography**: Inter exclusively. Headlines: 700/600 weight hierarchy with tight line-heights (1.1–1.2). Body: 400 weight (1.4 line-height). Labels: 600 weight, 12px, 0.02em letter-spacing
- **Typography scale**: Headline LG 28px/700, Headline MD 20px/600, Body LG 16px/400, Body MD 14px/400, Label SM 12px/600
- **Interaction**: Color-shift hover/active states (darker background on hover, no translate effects). Focus: border turns primary green + subtle shadow
- **Status pills**: Tonal style — light tinted background + dark text of same hue, 4px radius
- **Sidebar**: Active nav item indicated by 4px vertical green bar on leading edge + primary-container background
- **Spacing**: 8px rhythm. 4px/8px for grouping, 16px gutters, 24px margins

## Conversation

A threaded exchange of messages between a contact and one or more agents on a channel. Has status (open, pending, escalated, resolved, closed) and handled-by (ai, human, hybrid).
_Avoid_: Thread, ticket, chat.

## Contact

A person or organization that communicates via one or more channels. Identified by ChannelIdentities (phone number, page-scoped ID, etc.). Displayed as "Customers" in the Desk UI.
_Avoid_: Customer (as domain term), client, user (those mean Member, a Tenant's staff).

## Template

A message template synced with Meta. Has status (draft, pending, approved, rejected), language, body, and parameters.
_Avoid_: Saved reply, quick reply.

## Member

A user account within a Tenant who can log in to the Desk and handle conversations.
_Avoid_: Agent, user, staff.

## Tenant

An organization that owns channels, contacts, conversations, and members. Multi-tenancy is enforced at the API level via JWT claims.
_Avoid_: Organization, workspace, account.