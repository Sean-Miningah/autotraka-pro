import { writable, get } from 'svelte/store';
import { createApiClient, ClientError } from '$lib/api/client';
import { getGatewayUrl } from '$lib/api/config';
import { auth } from './auth';

export interface EnrichedConversation {
	id: string;
	tenant_id: string;
	contact_id: string;
	status: string;
	assigned_member_id: string | null;
	handled_by: string;
	previous_conversation_id: string | null;
	created_at: string;
	updated_at: string;
	contact_name: string;
	channel_type: string;
	last_message: string;
	last_message_at: string;
	unread_count: number;
}

interface ConversationsState {
	conversations: EnrichedConversation[];
	loading: boolean;
	error: string | null;
	statusFilter: string | null;
	handledByFilter: string | null;
	hasMore: boolean;
	offset: number;
}

const PAGE_SIZE = 20;

function createConversationsStore() {
	const store = writable<ConversationsState>({
		conversations: [],
		loading: false,
		error: null,
		statusFilter: null,
		handledByFilter: null,
		hasMore: true,
		offset: 0
	});

	function getApi() {
		const token = auth.getAccessToken();
		const baseUrl = getGatewayUrl();
		return createApiClient({
			baseUrl,
			getAccessToken: () => token,
			onRefresh: async () => {
				const newToken = auth.getAccessToken() ?? '';
				return newToken;
			}
		});
	}

	function buildQueryParams(state: ConversationsState, offset: number): string {
		const params = new URLSearchParams();
		params.set('limit', String(PAGE_SIZE));
		params.set('offset', String(offset));
		if (state.statusFilter) params.set('status', state.statusFilter);
		if (state.handledByFilter) params.set('handled_by', state.handledByFilter);
		return params.toString();
	}

	function parseLastMessage(raw: string): string {
		if (!raw) return '';
		try {
			const parsed = JSON.parse(raw);
			return parsed.text ?? parsed.body ?? raw;
		} catch {
			return raw.length > 80 ? raw.slice(0, 80) + '...' : raw;
		}
	}

	return {
		subscribe: store.subscribe,

		async fetchInitial() {
			const state = get(store);
			store.set({ ...state, loading: true, error: null, offset: 0 });
			try {
				const api = getApi();
				const qs = buildQueryParams({ ...state, offset: 0 }, 0);
				const result = await api.get(`/api/v1/conversations?${qs}`);
				const data = result.data as { conversations: EnrichedConversation[]; pagination: { total: number; limit: number; offset: number } };
				store.set({
					conversations: data.conversations,
					loading: false,
					error: null,
					statusFilter: state.statusFilter,
					handledByFilter: state.handledByFilter,
					hasMore: data.conversations.length < (data.pagination?.total ?? 0),
					offset: data.conversations.length
				});
			} catch (err) {
				const message = err instanceof ClientError ? err.message : 'Failed to load conversations';
				store.update(s => ({ ...s, loading: false, error: message }));
			}
		},

		async loadMore() {
			const state = get(store);
			if (state.loading || !state.hasMore) return;
			store.update(s => ({ ...s, loading: true }));
			try {
				const api = getApi();
				const qs = buildQueryParams(state, state.offset);
				const result = await api.get(`/api/v1/conversations?${qs}`);
				const data = result.data as { conversations: EnrichedConversation[]; pagination: { total: number } };
				store.update(s => ({
					conversations: [...s.conversations, ...data.conversations],
					loading: false,
					error: null,
					statusFilter: s.statusFilter,
					handledByFilter: s.handledByFilter,
					hasMore: s.conversations.length + data.conversations.length < (data.pagination?.total ?? 0),
					offset: s.conversations.length + data.conversations.length
				}));
			} catch (err) {
				const message = err instanceof ClientError ? err.message : 'Failed to load more conversations';
				store.update(s => ({ ...s, loading: false, error: message }));
			}
		},

		setStatusFilter(status: string | null) {
			store.update(s => ({ ...s, statusFilter: status }));
		},

		setHandledByFilter(handledBy: string | null) {
			store.update(s => ({ ...s, handledByFilter: handledBy }));
		},

		handleWebSocketEvent(event: { type: string; payload: Record<string, unknown> }) {
			store.update(s => {
				const convs = [...s.conversations];
				const convId = event.payload?.conversation_id as string | undefined;

				if (event.type === 'new_message' || event.type === 'conversation_updated') {
					if (!convId) return s;
					const idx = convs.findIndex(c => c.id === convId);
					if (idx !== -1) {
						const existing = convs[idx];
						convs[idx] = {
							...existing,
							updated_at: event.payload.updated_at as string ?? existing.updated_at,
							status: (event.payload.status as string) ?? existing.status,
							handled_by: (event.payload.handled_by as string) ?? existing.handled_by
						};
						const [moved] = convs.splice(idx, 1);
						convs.unshift(moved);
					} else {
						s.hasMore = true;
					}
				}

				if (event.type === 'new_message' && convId) {
					const idx = convs.findIndex(c => c.id === convId);
					if (idx !== -1) {
						convs[idx] = {
							...convs[idx],
							last_message: (event.payload.content as string) ?? convs[idx].last_message,
							unread_count: convs[idx].unread_count + 1
						};
					}
				}

				if (event.type === 'escalation' && convId) {
					const idx = convs.findIndex(c => c.id === convId);
					if (idx !== -1) {
						convs[idx] = { ...convs[idx], status: 'escalated' };
						const [moved] = convs.splice(idx, 1);
						convs.unshift(moved);
					}
				}

				return { ...s, conversations: convs };
			});
		},

		parseLastMessage
	};
}

export const conversations = createConversationsStore();