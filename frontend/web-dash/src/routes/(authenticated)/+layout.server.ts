import { redirect } from '@sveltejs/kit';
import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = async ({ cookies, url }: { cookies: any; url: any }) => {
	const refreshToken: string | undefined = cookies.get('refresh_token');
	const accessToken: string | null = url.searchParams.get('access_token');

	if (!refreshToken && !accessToken) {
		throw redirect(302, `/auth/login?redirectTo=${encodeURIComponent(url.pathname)}`);
	}

	return {};
};