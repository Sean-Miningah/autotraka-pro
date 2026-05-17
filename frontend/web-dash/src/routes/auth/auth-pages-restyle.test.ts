import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));

function readPage(route: string): string {
	return readFileSync(resolve(__dirname, `../../routes/${route}/+page.svelte`), 'utf-8');
}

describe('auth pages restyle', () => {
	const login = readPage('auth/login');
	const register = readPage('auth/register');

	describe('tonal error alerts', () => {
		it('login error has no border class, uses tonal error container styling', () => {
			const errorDiv = login.match(/<div[^>]*error[^>]*>.*?<\/div>/s)?.[0] ?? '';
			expect(errorDiv).not.toContain('border ');
			expect(errorDiv).not.toContain('border-error');
			expect(errorDiv).toContain('bg-error/10');
			expect(errorDiv).toContain('text-on-error-container');
		});

		it('register error has no border class, uses tonal error container styling', () => {
			const errorDiv = register.match(/<div[^>]*error[^>]*>.*?<\/div>/s)?.[0] ?? '';
			expect(errorDiv).not.toContain('border ');
			expect(errorDiv).not.toContain('border-error');
			expect(errorDiv).toContain('bg-error/10');
			expect(errorDiv).toContain('text-on-error-container');
		});
	});

describe('form labels use label-sm typography', () => {
		it('login labels use text-xs font-semibold tracking-wide instead of text-sm font-semibold', () => {
			const labels = login.match(/<label[^>]*>.*?<\/label>/gs) ?? [];
			for (const label of labels) {
				expect(label).not.toContain('text-sm');
				expect(label).toContain('text-xs');
				expect(label).toContain('tracking-wide');
			}
		});

		it('register labels use text-xs font-semibold tracking-wide instead of text-sm font-semibold', () => {
			const labels = register.match(/<label[^>]*>.*?<\/label>/gs) ?? [];
			for (const label of labels) {
				expect(label).not.toContain('text-sm');
				expect(label).toContain('text-xs');
				expect(label).toContain('tracking-wide');
			}
		});
	});

	describe('no neo-brutalist patterns remain', () => {
		it('login page has no old patterns', () => {
			expect(login).not.toContain('border-2');
			expect(login).not.toContain('shadow-[4px');
			expect(login).not.toContain('shadow-[2px_2px');
			expect(login).not.toContain('border-text');
			expect(login).not.toContain('dark:');
			expect(login).not.toContain('bg-base');
			expect(login).not.toContain('shadow-text');
			expect(login).not.toContain('translate-x');
			expect(login).not.toContain('translate-y');
		});

		it('register page has no old patterns', () => {
			expect(register).not.toContain('border-2');
			expect(register).not.toContain('shadow-[4px');
			expect(register).not.toContain('shadow-[2px_2px');
			expect(register).not.toContain('border-text');
			expect(register).not.toContain('dark:');
			expect(register).not.toContain('bg-base');
			expect(register).not.toContain('shadow-text');
			expect(register).not.toContain('translate-x');
			expect(register).not.toContain('translate-y');
		});
	});
});