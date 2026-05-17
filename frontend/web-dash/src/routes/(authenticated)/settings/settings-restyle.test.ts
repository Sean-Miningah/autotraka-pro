import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const page = readFileSync(resolve(__dirname, './+page.svelte'), 'utf-8');

describe('settings page restyle', () => {
	it('has no dark mode toggle', () => {
		expect(page).not.toContain("theme.setMode('dark')");
		expect(page).not.toContain("'dark'");
		expect(page).not.toContain('dark:');
	});

	it('uses bg-surface background', () => {
		expect(page).toContain('bg-surface');
	});

	it('uses text-on-surface-variant for body text', () => {
		expect(page).toContain('text-on-surface-variant');
	});

	it('logout uses danger variant', () => {
		expect(page).toContain('variant="danger"');
	});

	it('has no neo-brutalist classes', () => {
		expect(page).not.toContain('border-2');
		expect(page).not.toContain('shadow-[4px');
		expect(page).not.toContain('border-text');
	});
});