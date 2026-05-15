import { getGatewayUrl } from '$lib/api/config';
import { redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ cookies }) => {
	const refreshToken = cookies.get('refresh_token');
	if (refreshToken) {
		throw redirect(302, '/inbox');
	}
	return {};
};

export const actions: Actions = {
	register: async ({ request, cookies }) => {
		const formData = await request.formData();
		const tenantName = formData.get('tenant_name') as string;
		const email = formData.get('email') as string;
		const password = formData.get('password') as string;

		const response = await fetch(`${getGatewayUrl()}/api/v1/auth/register`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ tenant_name: tenantName, email, password })
		});

		if (!response.ok) {
			const body = await response.json();
			return { success: false, error: body.error || 'Registration failed' };
		}

		const body = await response.json();
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
	}
};