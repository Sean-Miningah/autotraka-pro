import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const page = readFileSync(resolve(__dirname, './+page.svelte'), 'utf-8');

describe('inbox page restyle', () => {
	describe('header', () => {
		it('uses bg-surface-container with 1px outline-variant border', () => {
			const header = page.match(/<div[^>]*border-b[^>]*>/s)?.[0] ?? '';
			expect(header).toContain('bg-surface-container');
			expect(header).toContain('border-outline-variant');
			expect(header).not.toContain('border-b-2');
			expect(header).not.toContain('border-text');
		});
	});

	describe('status filter tabs', () => {
		it('active tab uses underline indicator with text-primary, no shadow/translate', () => {
			expect(page).toContain('border-b-2');
			expect(page).toContain('border-primary');
			expect(page).not.toContain('shadow-[2px');
			expect(page).not.toContain('translate-x');
			expect(page).not.toContain('translate-y');
		});

		it('inactive tab uses text-on-surface-variant', () => {
			expect(page).toContain('text-on-surface-variant');
		});
	});

	describe('conversation rows', () => {
		it('uses 1px outline-variant separators, no border-text on rows', () => {
			expect(page).toContain('border-outline-variant');
			expect(page).not.toContain('border-text/10');
		});

		it('hover state uses bg-surface-container-high', () => {
			expect(page).toContain('hover:bg-surface-container-high');
		});

		it('timestamps and previews use text-on-surface-variant', () => {
			expect(page).toContain('text-on-surface-variant');
		});

		it('has active conversation indicator with 4px green bar', () => {
			expect(page).toContain('border-l-[4px]');
			expect(page).toContain('border-primary');
		});
	});

	describe('filter button', () => {
		it('uses outline variant Button component', () => {
			expect(page).toContain('variant="outline"');
		});
	});

	describe('badge', () => {
		it('unread count uses tonal variant', () => {
			expect(page).toContain('variant="tonal"');
		});
	});

	describe('no old patterns', () => {
		it('has no neo-brutalist classes', () => {
			expect(page).not.toContain('dark:');
			expect(page).not.toContain('shadow-[4px');
			expect(page).not.toContain('shadow-text');
			expect(page).not.toContain('bg-base');
		});
	});
});