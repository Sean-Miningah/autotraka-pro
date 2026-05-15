import { getGatewayUrl } from '$lib/api/config';
import type { Actions, PageServerLoad } from './$types';
import type { Cookies } from '@sveltejs/kit';

export const load: PageServerLoad = async ({ cookies, url }) => {
	const refreshToken = cookies.get('refresh_token');
	const accessToken = url.searchParams.get('access_token');

	if (refreshToken && !accessToken) {
		try {
			const response = await fetch(`${getGatewayUrl()}/api/v1/auth/refresh`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ refresh_token: refreshToken })
			});
			if (response.ok) {
				const body = await response.json();
				return { access_token: body.data.access_token };
			}
		} catch {
			// Continue to login page
		}
		cookies.delete('refresh_token', { path: '/' });
	}

	return {};
};

async function doLogin(email: string, password: string, tenantId: string, cookies: Cookies) {
	const response = await fetch(`${getGatewayUrl()}/api/v1/auth/login`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ email, password, tenant_id: tenantId })
	});

	let body: any = {};
	const text = await response.text();
	try { body = JSON.parse(text); } catch { body = { error: text || `HTTP ${response.status}` }; }

	if (!response.ok) {
		return { success: false, error: body.error || 'Login failed' };
	}

	const data = body.data as { access_token: string; refresh_token: string; expires_in: number };

	cookies.set('refresh_token', data.refresh_token, {
		path: '/',
		httpOnly: true,
		secure: true,
		sameSite: 'strict',
		maxAge: data.expires_in
	});

	return { success: true, access_token: data.access_token };
}

export const actions: Actions = {
	login: async ({ request, cookies }) => {
		try {
			const formData = await request.formData();
			const email = formData.get('email') as string;
			const password = formData.get('password') as string;
			const tenantId = formData.get('tenant_id') as string;

			// tenant_id provided - direct login
			if (tenantId) {
				return await doLogin(email, password, tenantId, cookies);
			}

			// No tenant_id - do lookup first
			const lookupResponse = await fetch(`${getGatewayUrl()}/api/v1/auth/tenants?email=${encodeURIComponent(email)}`);

			let lookupBody: any = {};
			const text = await lookupResponse.text();
			try { lookupBody = JSON.parse(text); } catch { lookupBody = { error: text || `HTTP ${lookupResponse.status}` }; }

			if (!lookupResponse.ok) {
				if (lookupResponse.status === 404) {
					return { success: false, error: 'No account found for this email' };
				}
				return { success: false, error: lookupBody.error || 'Lookup failed' };
			}

			const tenants = lookupBody.data as { tenant_id: string; tenant_name: string }[];

			if (tenants.length === 0) {
				return { success: false, error: 'No account found for this email' };
			}

			if (tenants.length === 1) {
				return await doLogin(email, password, tenants[0].tenant_id, cookies);
			}

			// Multiple tenants - ask user to pick
			return { success: false, needsTenantSelection: true, tenants };
		} catch (e) {
			console.error('Login flow error:', e);
			return { success: false, error: 'Connection error. Please check that the backend is running.' };
		}
	}
};
