import { describe, it, expect } from 'vitest';
import { readFileSync, existsSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const authDir = resolve(__dirname, '../../routes/(authenticated)');

describe('#43: templates route removed', () => {
	it('templates route directory does not exist', () => {
		expect(existsSync(resolve(authDir, 'templates'))).toBe(false);
	});

	it('sidebar layout has no templates nav item', () => {
		const layout = readFileSync(resolve(authDir, '+layout.svelte'), 'utf-8');
		expect(layout).not.toContain("label: 'Templates'");
		expect(layout).not.toContain("href: '/templates'");
	});
});