import { writable, get } from 'svelte/store';
import { browser } from '$app/environment';
import { getGatewayUrl } from '$lib/api/config';
import { auth } from './auth';
import { conversations } from './conversations';

interface WsState {
	connected: boolean;
	error: string | null;
}

function createWebSocketStore() {
	const store = writable<WsState>({ connected: false, error: null });
	let ws: WebSocket | null = null;
	let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
	let reconnectAttempts = 0;

	function connect() {
		if (!browser) return;

		const token = auth.getAccessToken();
		if (!token) return;

		const baseUrl = getGatewayUrl().replace(/^http/, 'ws');
		const url = `${baseUrl}/api/v1/ws?token=${encodeURIComponent(token)}`;

		try {
			ws = new WebSocket(url);
		} catch {
			scheduleReconnect();
			return;
		}

		ws.onopen = () => {
			store.set({ connected: true, error: null });
			reconnectAttempts = 0;
		};

		ws.onmessage = (event) => {
			try {
				const data = JSON.parse(event.data);
				if (data.type && data.payload) {
					conversations.handleWebSocketEvent({
						type: data.type,
						payload: data.payload
					});
				}
			} catch {
				// ignore malformed messages
			}
		};

		ws.onclose = () => {
			store.set({ connected: false, error: null });
			ws = null;
			scheduleReconnect();
		};

		ws.onerror = () => {
			store.set({ connected: false, error: 'WebSocket error' });
		};
	}

	function scheduleReconnect() {
		if (reconnectTimer) clearTimeout(reconnectTimer);
		if (reconnectAttempts >= 10) return;
		const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
		reconnectAttempts++;
		reconnectTimer = setTimeout(connect, delay);
	}

	function disconnect() {
		if (reconnectTimer) {
			clearTimeout(reconnectTimer);
			reconnectTimer = null;
		}
		if (ws) {
			ws.onclose = null;
			ws.close();
			ws = null;
		}
		store.set({ connected: false, error: null });
		reconnectAttempts = 10;
	}

	return {
		subscribe: store.subscribe,
		connect,
		disconnect
	};
}

export const ws = createWebSocketStore();