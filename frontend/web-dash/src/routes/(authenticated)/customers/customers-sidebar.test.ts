import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));

function read(name: string): string {
	return readFileSync(resolve(__dirname, '../../../lib/ui', name), 'utf-8');
}

describe('CustomersSidebar', () => {
	const src = read('CustomersSidebar.svelte');

	it('has 280px fixed width with right border', () => {
		expect(src).toContain('w-[280px]');
		expect(src).toContain('border-r');
		expect(src).toContain('border-outline-variant');
	});

	it('contains a search input', () => {
		expect(src).toContain('type="text"');
		expect(src).toContain('placeholder');
	});

	it('has scrollable customer list', () => {
		expect(src).toContain('overflow-y-auto');
	});

	it('shows avatar, name, channel badge, and last message for each item', () => {
		expect(src).toContain('customer.name');
		expect(src).toContain('customer.lastMessage');
		expect(src).toContain('rounded-full');
	});

	it('highlights active customer with primary green left border', () => {
		expect(src).toContain('border-l-[4px]');
		expect(src).toContain('border-l-primary');
		expect(src).toContain('bg-primary-container');
	});

	it('has hover state on items', () => {
		expect(src).toContain('hover:bg-surface-container-high');
	});
});
