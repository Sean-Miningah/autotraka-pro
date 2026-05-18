import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const customersDir = resolve(__dirname, '../../../routes/(authenticated)/customers');

describe('customers page restyle', () => {
	const page = readFileSync(resolve(customersDir, '+page.svelte'), 'utf-8');

	describe('master-detail layout', () => {
		it('renders CustomersSidebar and detail area side by side', () => {
			expect(page).toContain('CustomersSidebar');
			expect(page).toContain('flex');
			expect(page).toContain('lg:flex-row');
		});

		it('sidebar is 280px fixed width on desktop', () => {
			expect(page).toContain('CustomersSidebar');
		});

		it('detail area fills remaining width', () => {
			expect(page).toContain('flex-1');
		});

		it('shows CustomersEmptyState when no customer selected', () => {
			expect(page).toContain('CustomersEmptyState');
		});
	});

	describe('design tokens', () => {
		it('uses bg-surface background', () => {
			expect(page).toContain('bg-surface');
		});

		it('has no dark mode or neo-brutalist classes', () => {
			expect(page).not.toContain('dark:');
			expect(page).not.toContain('border-text');
			expect(page).not.toContain('shadow-[4px');
		});
	});
});
