import { writable, get } from 'svelte/store';

export interface TenantEntry {
	tenant_id: string;
	tenant_name: string;
}

interface AuthState {
	isAuthenticated: boolean;
	accessToken: string | null;
	user: { email: string; role: string } | null;
	tenant: TenantEntry | null;
}

export class AuthError extends Error {
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

interface AuthDeps {
	api: {
		post: (path: string, body: unknown) => Promise<{ data: unknown }>;
		get: (path: string) => Promise<{ data: unknown }>;
		patch: (path: string, body: unknown) => Promise<{ data: unknown }>;
		delete: (path: string) => Promise<{ data: unknown }>;
	};
}

export function createAuthStore(deps: AuthDeps) {
	const store = writable<AuthState>({
		isAuthenticated: false,
		accessToken: null,
		user: null,
		tenant: null
	});

	return {
		subscribe: store.subscribe,

		login: async (email: string, password: string, tenantId: string) => {
			const result = await deps.api.post('/api/v1/auth/login', {
				email,
				password,
				tenant_id: tenantId
			});

			const data = result.data as {
				access_token: string;
				refresh_token: string;
				expires_in: number;
			};

			store.set({
				isAuthenticated: true,
				accessToken: data.access_token,
				user: { email, role: 'member' },
				tenant: { tenant_id: tenantId, tenant_name: '' }
			});
		},

		logout: () => {
			store.set({
				isAuthenticated: false,
				accessToken: null,
				user: null,
				tenant: null
			});
		},

		selectTenant: (tenant: TenantEntry) => {
			const current = get(store);
			store.set({ ...current, tenant });
		},

		getAccessToken: () => get(store).accessToken
	};
}