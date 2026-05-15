import { getGatewayUrl } from '$lib/api/config';
import type { Actions, PageServerLoad } from './$types';

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

export const actions: Actions = {
	login: async ({ request, cookies }) => {
		const formData = await request.formData();
		const email = formData.get('email') as string;
		const password = formData.get('password') as string;
		const tenantId = formData.get('tenant_id') as string;

		const response = await fetch(`${getGatewayUrl()}/api/v1/auth/login`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ email, password, tenant_id: tenantId })
		});

		if (!response.ok) {
			const body = await response.json();
			return { success: false, error: body.error || 'Login failed' };
		}

		const body = await response.json();
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
};