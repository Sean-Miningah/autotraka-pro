import { describe, it, expect } from 'vitest';
import { readFileSync, existsSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const authDir = resolve(__dirname, '../../routes/(authenticated)');

describe('#45: copilots route stub', () => {
	const page = readFileSync(resolve(authDir, 'copilots/+page.svelte'), 'utf-8');

	it('heading says Copilots', () => {
		expect(page).toContain('Copilots');
	});

	it('uses text-headline-lg and text-on-surface', () => {
		expect(page).toContain('text-[28px]');
		expect(page).toContain('font-bold');
		expect(page).toContain('leading-tight');
		expect(page).toContain('text-on-surface');
	});

	it('uses text-on-surface-variant for subtitle', () => {
		expect(page).toContain('text-on-surface-variant');
	});

	it('uses bg-surface background', () => {
		expect(page).toContain('bg-surface');
	});

	it('has no dark or brutal patterns', () => {
		expect(page).not.toContain('dark:');
		expect(page).not.toContain('border-text');
	});
});

describe('#48: dashboards route stub', () => {
	const page = readFileSync(resolve(authDir, 'dashboards/+page.svelte'), 'utf-8');

	it('heading says Dashboards', () => {
		expect(page).toContain('Dashboards');
	});

	it('uses text-headline-lg and text-on-surface', () => {
		expect(page).toContain('text-[28px]');
		expect(page).toContain('text-on-surface');
	});

	it('uses text-on-surface-variant for body text', () => {
		expect(page).toContain('text-on-surface-variant');
	});

	it('uses bg-surface background', () => {
		expect(page).toContain('bg-surface');
	});

	it('has metric card slots', () => {
		expect(page).toContain('Open Conversations');
		expect(page).toContain('Messages Today');
		expect(page).toContain('Avg Response Time');
	});

	it('uses card-level styling tokens', () => {
		expect(page).toContain('surface-container-lowest');
		expect(page).toContain('shadow-elevation-1');
	});

	it('root redirect points to /dashboards', () => {
		const redirect = readFileSync(resolve(authDir, '../+page.server.ts'), 'utf-8');
		expect(redirect).toContain('/dashboards');
		expect(redirect).not.toContain('/inbox');
	});
});