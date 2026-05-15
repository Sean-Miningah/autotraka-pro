import { describe, it, expect } from 'vitest';
import { get } from 'svelte/store';
import { createAuthStore } from './auth';

describe('auth store', () => {
	it('starts unauthenticated', () => {
		const store = createAuthStore();
		const state = get(store);
		expect(state.isAuthenticated).toBe(false);
		expect(state.user).toBeNull();
		expect(state.accessToken).toBeNull();
	});

	it('setToken marks as authenticated with the token', () => {
		const store = createAuthStore();
		store.setToken('at_123');

		const state = get(store);
		expect(state.isAuthenticated).toBe(true);
		expect(state.accessToken).toBe('at_123');
	});

	it('logout clears user, accessToken, and isAuthenticated', () => {
		const store = createAuthStore();
		store.setToken('at_123');
		expect(get(store).isAuthenticated).toBe(true);

		store.logout();

		const state = get(store);
		expect(state.isAuthenticated).toBe(false);
		expect(state.accessToken).toBeNull();
		expect(state.user).toBeNull();
	});

	it('selectTenant stores the selected tenant', () => {
		const store = createAuthStore();
		store.setToken('at_123');

		store.selectTenant({
			tenant_id: '22222222-2222-2222-2222-222222222222',
			tenant_name: 'Beta Inc'
		});

		const state = get(store);
		expect(state.tenant?.tenant_id).toBe('22222222-2222-2222-2222-222222222222');
		expect(state.tenant?.tenant_name).toBe('Beta Inc');
	});

	it('getAccessToken returns the current access token', () => {
		const store = createAuthStore();
		expect(store.getAccessToken()).toBeNull();

		store.setToken('at_456');
		expect(store.getAccessToken()).toBe('at_456');
	});
});