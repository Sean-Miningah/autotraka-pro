import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { createApiClient } from './client';

describe('API client', () => {
	let client: ReturnType<typeof createApiClient>;
	let fetchMock: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		fetchMock = vi.fn();
		vi.stubGlobal('fetch', fetchMock);
		client = createApiClient({
			baseUrl: 'http://localhost:8080',
			getAccessToken: () => 'test-access-token',
			onRefresh: vi.fn().mockResolvedValue('new-access-token')
		});
	});

	afterEach(() => {
		vi.restoreAllMocks();
	});

	it('injects bearer token into GET requests', async () => {
		fetchMock.mockResolvedValueOnce(
			new Response(JSON.stringify({ data: { id: '1' } }), { status: 200, headers: { 'Content-Type': 'application/json' } })
		);

		const result = await client.get('/api/v1/conversations');
		expect(fetchMock).toHaveBeenCalledWith(
			'http://localhost:8080/api/v1/conversations',
			expect.objectContaining({
				headers: expect.objectContaining({ Authorization: 'Bearer test-access-token' })
			})
		);
	});

	it('refreshes token on 401 and retries the request', async () => {
		const onRefresh = vi.fn().mockResolvedValue('new-access-token');

		client = createApiClient({
			baseUrl: 'http://localhost:8080',
			getAccessToken: () => 'expired-token',
			onRefresh
		});

		fetchMock
			.mockResolvedValueOnce(new Response(JSON.stringify({ error: 'unauthorized' }), { status: 401 }))
			.mockResolvedValueOnce(
				new Response(JSON.stringify({ data: { id: '1' } }), { status: 200, headers: { 'Content-Type': 'application/json' } })
			);

		const result = await client.get('/api/v1/conversations');
		expect(onRefresh).toHaveBeenCalledOnce();
		expect(fetchMock).toHaveBeenCalledTimes(2);
		expect(result.data).toEqual({ id: '1' });
	});

	it('throws on non-401 error responses', async () => {
		fetchMock.mockResolvedValueOnce(
			new Response(JSON.stringify({ error: 'not found' }), { status: 404 })
		);

		await expect(client.get('/api/v1/conversations/missing')).rejects.toThrow('not found');
	});

	it('POSTs with JSON body and auth header', async () => {
		fetchMock.mockResolvedValueOnce(
			new Response(JSON.stringify({ data: { id: '2' } }), { status: 201, headers: { 'Content-Type': 'application/json' } })
		);

		await client.post('/api/v1/auth/login', { email: 'test@test.com', password: 'pass' });
		expect(fetchMock).toHaveBeenCalledWith(
			'http://localhost:8080/api/v1/auth/login',
			expect.objectContaining({
				method: 'POST',
				headers: expect.objectContaining({
					Authorization: 'Bearer test-access-token',
					'Content-Type': 'application/json'
				})
			})
		);
	});
});