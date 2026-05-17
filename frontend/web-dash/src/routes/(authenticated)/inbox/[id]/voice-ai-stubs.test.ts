import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const page = readFileSync(resolve(__dirname, './+page.svelte'), 'utf-8');

describe('#46: voice call and AI summary stubs', () => {
	it('AI summary panel renders with collapsible toggle', () => {
		expect(page).toContain('AI Summary');
		expect(page).toContain('showAiSummary');
	});

	it('AI summary uses surface-container bg and text-on-surface-variant', () => {
		expect(page).toContain('bg-surface-container');
		expect(page).toContain('text-on-surface-variant');
	});

	it('voice call button uses outline variant', () => {
		expect(page).toContain('variant="outline"');
	});

	it('uses new design tokens, no neo-brutalist', () => {
		expect(page).toContain('border-outline-variant');
		expect(page).not.toContain('dark:');
		expect(page).not.toContain('shadow-[4px');
		expect(page).not.toContain('border-text');
	});

	it('uses bg-surface for page background', () => {
		expect(page).toContain('bg-surface');
	});
});