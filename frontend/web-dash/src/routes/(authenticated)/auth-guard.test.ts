import { describe, it, expect } from 'vitest';

// The layout guard logic: if no refresh_token cookie and no access_token
// query param, redirect to /auth/login. Otherwise, proceed.
function shouldRedirectToLogin(cookies: { get: (name: string) => string | undefined }, searchParams: { get: (key: string) => string | null }): boolean {
	const refreshToken = cookies.get('refresh_token');
	const accessToken = searchParams.get('access_token');
	return !refreshToken && !accessToken;
}

describe('authenticated layout guard logic', () => {
	it('redirects when no refresh token and no access token', () => {
		const cookies = { get: () => undefined };
		const searchParams = { get: () => null };
		expect(shouldRedirectToLogin(cookies, searchParams)).toBe(true);
	});

	it('does not redirect when refresh token exists', () => {
		const cookies = { get: (name: string) => name === 'refresh_token' ? 'some-token' : undefined };
		const searchParams = { get: () => null };
		expect(shouldRedirectToLogin(cookies, searchParams)).toBe(false);
	});

	it('does not redirect when access token in URL params', () => {
		const cookies = { get: () => undefined };
		const searchParams = { get: (key: string) => key === 'access_token' ? 'at_123' : null };
		expect(shouldRedirectToLogin(cookies, searchParams)).toBe(false);
	});

	it('does not redirect when both exist', () => {
		const cookies = { get: (name: string) => name === 'refresh_token' ? 'rt_456' : undefined };
		const searchParams = { get: (key: string) => key === 'access_token' ? 'at_123' : null };
		expect(shouldRedirectToLogin(cookies, searchParams)).toBe(false);
	});
});