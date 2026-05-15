import { redirect } from '@sveltejs/kit';
import type { LayoutServerLoad } from './$types';
import { getGatewayUrl } from '$lib/api/config';

export const load: LayoutServerLoad = async ({ cookies, url, fetch }) => {
	const refreshToken: string | undefined = cookies.get('refresh_token');
	const accessTokenParam: string | null = url.searchParams.get('access_token');

	if (!refreshToken && !accessTokenParam) {
		throw redirect(302, `/auth/login?redirectTo=${encodeURIComponent(url.pathname)}`);
	}

	let accessToken: string | null = accessTokenParam;

	if (refreshToken && !accessToken) {
		try {
			const response = await fetch(`${getGatewayUrl()}/api/v1/auth/refresh`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ refresh_token: refreshToken })
			});
			if (response.ok) {
				const body = await response.json();
				accessToken = body.data?.access_token ?? body.access_token ?? null;
			}
		} catch {
			// Continue without token — the client store will handle it
		}
	}

	return { accessToken };
};