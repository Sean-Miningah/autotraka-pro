export function getGatewayUrl(): string {
	// import.meta.env works in both Vite client and SSR contexts
	const viteUrl = import.meta.env.VITE_GATEWAY_URL;
	if (viteUrl) return viteUrl;

	// Fallback for direct Node env vars (Docker, deployed environments, etc.)
	const processUrl = process.env.GATEWAY_URL || process.env.VITE_GATEWAY_URL;
	if (processUrl) return processUrl;

	return 'http://localhost:8080';
}