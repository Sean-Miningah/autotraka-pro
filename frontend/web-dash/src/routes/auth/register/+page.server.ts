import { getGatewayUrl } from '$lib/api/config';
import { redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ cookies }) => {
	const refreshToken = cookies.get('refresh_token');
	if (refreshToken) {
		throw redirect(302, '/dashboards');
	}
	return {};
};

export const actions: Actions = {
	register: async ({ request, cookies }) => {
		try {
			const formData = await request.formData();
			const tenantName = formData.get('tenant_name') as string;
			const email = formData.get('email') as string;
			const password = formData.get('password') as string;

      const gatewayUrl = getGatewayUrl();
			console.log('[register] loginResponse', gatewayUrl)
			const response = await fetch(`${getGatewayUrl()}/api/v1/auth/register`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ tenant_name: tenantName, email, password })
			});

			let body: any = {};
			const text = await response.text();
			try { body = JSON.parse(text); } catch { body = { error: text || `HTTP ${response.status}` }; }

			if (!response.ok) {
				return { success: false, error: body.error || 'Registration failed' };
			}

			const data = body.data as { tenant_id: string; member_id: string; email: string; role: string };

			const loginResponse = await fetch(`${getGatewayUrl()}/api/v1/auth/login`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ email, password, tenant_id: data.tenant_id })
			});

			if (!loginResponse.ok) {
				return { success: true, needsLogin: true };
			}

			const loginBody = await loginResponse.json();
			const loginData = loginBody.data as { access_token: string; refresh_token: string; expires_in: number };

			cookies.set('refresh_token', loginData.refresh_token, {
				path: '/',
				httpOnly: true,
				secure: true,
				sameSite: 'strict',
				maxAge: loginData.expires_in
			});

			return { success: true, access_token: loginData.access_token };
		} catch (e) {
			console.error('Registration error:', e);
			return { success: false, error: 'Connection error. Please check that the backend is running.' };
		}
	}
};
