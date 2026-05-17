import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const layout = readFileSync(resolve(__dirname, './+layout.svelte'), 'utf-8');

describe('shell restyle', () => {
	describe('desktop sidebar', () => {
		it('uses 1px outline-variant right border', () => {
			expect(layout).toContain('border-r');
			expect(layout).toContain('border-outline-variant');
			expect(layout).not.toContain('border-r-2');
			expect(layout).not.toContain('border-text');
		});

		it('has correct nav items: Dashboards, Inbox, Customers, Analytics, Copilots, Settings', () => {
			expect(layout).toContain("label: 'Dashboards'");
			expect(layout).toContain("label: 'Inbox'");
			expect(layout).toContain("label: 'Customers'");
			expect(layout).toContain("label: 'Analytics'");
			expect(layout).toContain("label: 'Copilots'");
			expect(layout).toContain("label: 'Settings'");
			expect(layout).not.toContain("label: 'Templates'");
			expect(layout).not.toContain("label: 'Contacts'");
		});

		it('uses correct nav routes', () => {
			expect(layout).toContain("href: '/dashboards'");
			expect(layout).toContain("href: '/inbox'");
			expect(layout).toContain("href: '/customers'");
			expect(layout).toContain("href: '/analytics'");
			expect(layout).toContain("href: '/copilots'");
			expect(layout).toContain("href: '/settings'");
		});

		it('active nav has 4px vertical green bar on leading edge', () => {
			expect(layout).toContain('border-l-[4px]');
			expect(layout).toContain('border-primary');
			expect(layout).not.toContain('border-l-2');
		});

		it('active nav uses primary-container background', () => {
			expect(layout).toContain('bg-primary-container');
		});

		it('inactive nav uses on-surface-variant text with hover background', () => {
			expect(layout).toContain('text-on-surface-variant');
			expect(layout).toContain('hover:bg-surface-container-high');
		});

		it('sidebar background is surface-container', () => {
			const aside = layout.match(/<aside[^>]*>/s)?.[0] ?? '';
			expect(aside).toContain('bg-surface-container');
		});
	});

	describe('mobile bottom nav', () => {
		it('has Inbox, Customers, Dashboards, Copilots, Settings tabs', () => {
			expect(layout).toContain("label: 'Inbox'");
			expect(layout).toContain("label: 'Customers'");
			expect(layout).toContain("label: 'Dashboards'");
			expect(layout).toContain("label: 'Copilots'");
			expect(layout).toContain("label: 'Settings'");
			expect(layout).not.toContain("label: 'Profile'");
			expect(layout).not.toContain("label: 'My Stats'");
		});

		it('uses 1px outline-variant top border', () => {
			const mobileNav = layout.match(/fixed inset-x-0 bottom-0[^>]*>/s)?.[0] ?? '';
			expect(mobileNav).toContain('border-t');
			expect(mobileNav).toContain('border-outline-variant');
			expect(mobileNav).not.toContain('border-t-2');
		});

		it('active tab has green text (text-primary)', () => {
			expect(layout).toContain('text-primary');
		});

		it('inactive tabs use text-on-surface-variant', () => {
			expect(layout).toContain('text-on-surface-variant');
		});

		it('mobile nav background is surface-container', () => {
			const mobileNav = layout.match(/fixed inset-x-0 bottom-0[^>]*>/s)?.[0] ?? '';
			expect(mobileNav).toContain('bg-surface-container');
		});
	});

	describe('no old patterns', () => {
		it('has no neo-brutalist classes', () => {
			expect(layout).not.toContain('border-2');
			expect(layout).not.toContain('shadow-[4px');
			expect(layout).not.toContain('shadow-text');
			expect(layout).not.toContain('translate-x');
			expect(layout).not.toContain('translate-y');
		});

		it('has no dark mode variants', () => {
			expect(layout).not.toContain('dark:');
		});

		it('uses bg-surface for layout wrapper', () => {
			expect(layout).toContain('bg-surface');
			expect(layout).not.toContain('bg-base');
		});
	});
});