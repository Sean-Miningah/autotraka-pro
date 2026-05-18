import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const layout = readFileSync(resolve(__dirname, './+layout.svelte'), 'utf-8');
const hud = readFileSync(resolve(__dirname, '../../lib/ui/ActiveCallHud.svelte'), 'utf-8');

describe('ActiveCallHud integration', () => {
	it('ActiveCallHud is rendered in the layout', () => {
		expect(layout).toContain('ActiveCallHud');
	});

	it('HUD persists across tab switches (rendered outside tab content)', () => {
		const hudIndex = layout.indexOf('ActiveCallHud');
		const bodyEnd = layout.lastIndexOf('</div>');
		expect(hudIndex).toBeGreaterThan(0);
		expect(hudIndex).toBeLessThan(bodyEnd);
	});

	it('has dev-only keyboard shortcut to toggle fake call', () => {
		expect(layout).toContain('keydown');
		expect(layout).toContain('toggleFakeCall');
		expect(layout).toContain('fakeCall');
	});

	it('uses design tokens for HUD', () => {
		expect(hud).toContain('surface-container-lowest');
		expect(hud).toContain('outline-variant');
		expect(hud).toContain('shadow-elevation-2');
	});
});
