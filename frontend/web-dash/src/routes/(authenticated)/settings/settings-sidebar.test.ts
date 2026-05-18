import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));

function read(name: string): string {
	return readFileSync(resolve(__dirname, '../../../lib/ui', name), 'utf-8');
}

describe('SettingsSidebar', () => {
	const src = read('SettingsSidebar.svelte');

	it('has 280px fixed width with right border', () => {
		expect(src).toContain('w-[280px]');
		expect(src).toContain('border-r');
		expect(src).toContain('border-outline-variant');
	});

	it('lists 5 sections', () => {
		expect(src).toContain('Profile');
		expect(src).toContain('Notifications');
		expect(src).toContain('Team');
		expect(src).toContain('Channels');
		expect(src).toContain('Billing');
	});

	it('highlights active section with primary green left border', () => {
		expect(src).toContain('border-l-[4px]');
		expect(src).toContain('border-l-primary');
		expect(src).toContain('bg-primary-container');
	});

	it('has hover state on items', () => {
		expect(src).toContain('hover:bg-surface-container-high');
	});

	it('navigates on click', () => {
		expect(src).toContain('onclick');
		expect(src).toContain('onselect');
	});
});
