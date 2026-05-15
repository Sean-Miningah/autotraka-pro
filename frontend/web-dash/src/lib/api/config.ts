export function getGatewayUrl(): string {
	if (typeof window !== 'undefined' && import.meta.env.VITE_GATEWAY_URL) {
		return import.meta.env.VITE_GATEWAY_URL;
	}
	if (typeof window === 'undefined' && process.env.GATEWAY_URL) {
		return process.env.GATEWAY_URL;
	}
	return 'http://localhost:8080';
}