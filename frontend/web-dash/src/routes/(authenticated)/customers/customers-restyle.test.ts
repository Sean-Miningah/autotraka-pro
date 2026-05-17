import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const customersDir = resolve(__dirname, '../../../routes/(authenticated)/customers');

describe('customers page restyle', () => {
	const page = readFileSync(resolve(customersDir, '+page.svelte'), 'utf-8');

	it('heading says Customers not Contacts', () => {
		expect(page).toContain('Customers');
		expect(page).not.toContain('Contacts');
	});

	it('uses text-headline-lg token on heading', () => {
		expect(page).toContain('text-[28px]');
		expect(page).toContain('font-bold');
		expect(page).toContain('leading-tight');
	});

	it('uses text-on-surface for heading', () => {
		expect(page).toContain('text-on-surface');
	});

	it('uses text-on-surface-variant for subtitle', () => {
		expect(page).toContain('text-on-surface-variant');
	});

	it('uses bg-surface background', () => {
		expect(page).toContain('bg-surface');
	});

	it('has no dark mode or neo-brutalist classes', () => {
		expect(page).not.toContain('dark:');
		expect(page).not.toContain('border-text');
		expect(page).not.toContain('shadow-[4px');
	});
});