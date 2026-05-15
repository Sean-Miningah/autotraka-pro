import { describe, it, expect, beforeEach, vi } from 'vitest';
import { get } from 'svelte/store';
import { createAuthStore } from './auth';

describe('auth store', () => {
	let store: ReturnType<typeof createAuthStore>;
	let mockApi: ReturnType<typeof createMockApi>;

	function createMockApi() {
		return {
			post: vi.fn(),
			get: vi.fn(),
			patch: vi.fn(),
			delete: vi.fn()
		};
	}

	beforeEach(() => {
		mockApi = createMockApi();
		store = createAuthStore({ api: mockApi });
	});

	it('starts unauthenticated', () => {
		const state = get(store);
		expect(state.isAuthenticated).toBe(false);
		expect(state.user).toBeNull();
		expect(state.accessToken).toBeNull();
	});

	it('login sets user, accessToken, and tenant on success', async () => {
		mockApi.post.mockResolvedValueOnce({
			data: {
				access_token: 'at_123',
				refresh_token: 'rt_456',
				expires_in: 900
			}
		});

		await store.login('admin@acme.com', 'password123', '11111111-1111-1111-1111-111111111111');

		const state = get(store);
		expect(state.isAuthenticated).toBe(true);
		expect(state.accessToken).toBe('at_123');
		expect(mockApi.post).toHaveBeenCalledWith('/api/v1/auth/login', {
			email: 'admin@acme.com',
			password: 'password123',
			tenant_id: '11111111-1111-1111-1111-111111111111'
		});
	});

	it('logout clears user, accessToken, and isAuthenticated', async () => {
		mockApi.post.mockResolvedValueOnce({
			data: { access_token: 'at_123', refresh_token: 'rt_456', expires_in: 900 }
		});

		await store.login('admin@acme.com', 'password123', '11111111-1111-1111-1111-111111111111');
		expect(get(store).isAuthenticated).toBe(true);

		store.logout();

		const state = get(store);
		expect(state.isAuthenticated).toBe(false);
		expect(state.accessToken).toBeNull();
		expect(state.user).toBeNull();
	});

	it('selectTenant stores the selected tenant', async () => {
		mockApi.post.mockResolvedValueOnce({
			data: { access_token: 'at_123', refresh_token: 'rt_456', expires_in: 900 }
		});

		await store.login('admin@acme.com', 'password123', '11111111-1111-1111-1111-111111111111');

		store.selectTenant({
			tenant_id: '22222222-2222-2222-2222-222222222222',
			tenant_name: 'Beta Inc'
		});

		const state = get(store);
		expect(state.tenant?.tenant_id).toBe('22222222-2222-2222-2222-222222222222');
		expect(state.tenant?.tenant_name).toBe('Beta Inc');
	});

	it('login failure leaves store unauthenticated', async () => {
		mockApi.post.mockRejectedValueOnce({ status: 401, body: { error: 'invalid email or password' } });

		await expect(
			store.login('bad@example.com', 'wrong', '11111111-1111-1111-1111-111111111111')
		).rejects.toThrow();

		const state = get(store);
		expect(state.isAuthenticated).toBe(false);
		expect(state.accessToken).toBeNull();
	});
});