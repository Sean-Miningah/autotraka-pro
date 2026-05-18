import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));

function read(name: string): string {
	return readFileSync(resolve(__dirname, '../../../lib/ui', name), 'utf-8');
}

describe('InboxEmptyState', () => {
	const src = read('InboxEmptyState.svelte');

	it('displays "Select a conversation" as primary text', () => {
		expect(src).toContain('Select a conversation');
	});

	it('displays secondary guidance text', () => {
		expect(src).toContain('Choose a conversation from the list to start messaging');
	});

	it('uses text-on-surface-variant for muted text', () => {
		expect(src).toContain('text-on-surface-variant');
	});

	it('has a centered layout', () => {
		expect(src).toContain('items-center');
		expect(src).toContain('justify-center');
	});
});
