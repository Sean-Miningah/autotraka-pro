import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const page = readFileSync(resolve(__dirname, './+page.svelte'), 'utf-8');

describe('settings page restyle', () => {
	describe('master-detail layout', () => {
		it('renders SettingsSidebar and content area side by side', () => {
			expect(page).toContain('SettingsSidebar');
			expect(page).toContain('flex');
			expect(page).toContain('lg:flex-row');
		});

		it('content area fills remaining width', () => {
			expect(page).toContain('flex-1');
		});

		it('hides content sidebar on mobile', () => {
			expect(page).toContain('hidden');
			expect(page).toContain('lg:flex');
		});
	});

	describe('URL-driven sections', () => {
	it('reads section from URL path', () => {
		expect(page).toContain('$page.url.pathname');
	});

		it('defaults to profile when no section', () => {
			expect(page).toContain('profile');
			expect(page).toContain('||');
		});
	});

	describe('sections', () => {
		it('has profile section with edit form', () => {
			expect(page).toContain('Profile');
			expect(page).toContain('input');
		});

		it('has stub sections with Coming soon text', () => {
			expect(page).toContain('Coming soon');
			expect(page).toContain('Notifications');
			expect(page).toContain('Team');
			expect(page).toContain('Channels');
			expect(page).toContain('Billing');
		});
	});

	describe('design tokens', () => {
		it('uses bg-surface background', () => {
			expect(page).toContain('bg-surface');
		});

		it('has no dark mode or neo-brutalist classes', () => {
			expect(page).not.toContain('dark:');
			expect(page).not.toContain('border-2');
			expect(page).not.toContain('shadow-[4px');
			expect(page).not.toContain('border-text');
		});
	});
});
