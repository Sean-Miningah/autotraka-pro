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

export function createAuthStore() {
	const store = writable<AuthState>({
		isAuthenticated: false,
		accessToken: null,
		user: null,
		tenant: null
	});

	return {
		subscribe: store.subscribe,

		setToken(token: string) {
			const current = get(store);
			store.set({
				...current,
				isAuthenticated: true,
				accessToken: token
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

export const auth = createAuthStore();