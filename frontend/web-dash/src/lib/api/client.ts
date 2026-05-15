interface ApiClientConfig {
	baseUrl: string;
	getAccessToken: () => string | null;
	onRefresh?: () => Promise<string>;
}

interface ApiError extends Error {
	status: number;
	body: unknown;
}

export class ClientError extends Error {
	status: number;
	body: unknown;

	constructor(status: number, body: unknown) {
		const message = typeof body === 'object' && body !== null && 'error' in body
			? String((body as { error: string }).error)
			: `Request failed with status ${status}`;
		super(message);
		this.status = status;
		this.body = body;
	}
}

export function createApiClient(config: ApiClientConfig) {
	let refreshing: Promise<string> | null = null;

	async function request(path: string, options: RequestInit = {}): Promise<unknown> {
		const token = config.getAccessToken();
		const headers: Record<string, string> = {
			'Content-Type': 'application/json',
			...(options.headers as Record<string, string> ?? {})
		};
		if (token) {
			headers['Authorization'] = `Bearer ${token}`;
		}

		const response = await fetch(`${config.baseUrl}${path}`, {
			...options,
			headers
		});

		if (response.status === 401 && config.onRefresh) {
			if (!refreshing) {
				refreshing = config.onRefresh();
			}
			const newToken = await refreshing;
			refreshing = null;

			headers['Authorization'] = `Bearer ${newToken}`;
			const retryResponse = await fetch(`${config.baseUrl}${path}`, {
				...options,
				headers
			});

			if (!retryResponse.ok) {
				const retryBody = await retryResponse.json().catch(() => null);
				throw new ClientError(retryResponse.status, retryBody);
			}

			return retryResponse.json();
		}

		if (!response.ok) {
			const body = await response.json().catch(() => null);
			throw new ClientError(response.status, body);
		}

		return response.json();
	}

	return {
		get: (path: string) => request(path) as Promise<{ data: unknown }>,
		post: (path: string, body: unknown) =>
			request(path, { method: 'POST', body: JSON.stringify(body) }) as Promise<{ data: unknown }>,
		patch: (path: string, body: unknown) =>
			request(path, { method: 'PATCH', body: JSON.stringify(body) }) as Promise<{ data: unknown }>,
		delete: (path: string) =>
			request(path, { method: 'DELETE' }) as Promise<{ data: unknown }>
	};
}