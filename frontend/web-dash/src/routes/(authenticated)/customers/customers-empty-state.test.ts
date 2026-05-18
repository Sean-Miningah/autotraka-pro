import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));

function read(name: string): string {
	return readFileSync(resolve(__dirname, '../../../lib/ui', name), 'utf-8');
}

describe('CustomersEmptyState', () => {
	const src = read('CustomersEmptyState.svelte');

	it('displays "Select a customer" as primary text', () => {
		expect(src).toContain('Select a customer');
	});

	it('displays secondary guidance text', () => {
		expect(src).toContain('Choose a customer from the list to view details');
	});

	it('uses text-on-surface-variant for muted text', () => {
		expect(src).toContain('text-on-surface-variant');
	});

	it('has a centered layout', () => {
		expect(src).toContain('items-center');
		expect(src).toContain('justify-center');
	});
});
