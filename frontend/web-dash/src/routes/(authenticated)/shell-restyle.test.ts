import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const layout = readFileSync(resolve(__dirname, './+layout.svelte'), 'utf-8');
const tabBar = readFileSync(resolve(__dirname, '../../lib/ui/DesktopTabBar.svelte'), 'utf-8');
const topBar = readFileSync(resolve(__dirname, '../../lib/ui/GlobalTopBar.svelte'), 'utf-8');

describe('shell layout', () => {
	describe('desktop shell', () => {
		it('has no sidebar (<aside>) element', () => {
			expect(layout).not.toContain('<aside');
		});

		it('has no lg:pl-64 padding on main content', () => {
			expect(layout).not.toContain('lg:pl-64');
		});

		it('desktop container is a vertical flex column filling viewport', () => {
			expect(layout).toContain('lg:flex-col');
			expect(layout).toContain('lg:h-screen');
		});

		it('renders GlobalTopBar component', () => {
			expect(layout).toContain('<GlobalTopBar');
		});

		it('renders DesktopTabBar component', () => {
			expect(layout).toContain('<DesktopTabBar');
		});

		it('content area is scrollable and flex-1', () => {
			expect(layout).toContain('flex-1');
			expect(layout).toContain('overflow-auto');
		});
	});

	describe('GlobalTopBar', () => {
		it('has correct height (h-12 = 48px)', () => {
			expect(topBar).toContain('h-12');
		});

		it('logo reads "Auto" followed by "traka"', () => {
			expect(topBar).toContain('Auto');
			expect(topBar).toContain('traka');
			expect(topBar).toContain('text-primary');
		});

		it('logo button navigates to Dashboards tab via tabs store', () => {
			expect(topBar).toContain("tabs.openTab('dashboards')");
		});

		it('has search icon button', () => {
			expect(topBar).toContain('aria-label="Search"');
		});

		it('has notifications bell icon button', () => {
			expect(topBar).toContain('aria-label="Notifications"');
		});

		it('has user avatar button with initial', () => {
			expect(topBar).toContain('aria-label="User menu"');
			expect(topBar).toContain('rounded-[var(--radius-full)]');
			expect(topBar).toContain('userInitial');
		});
	});

	describe('DesktopTabBar', () => {
		it('renders tabs from the tab store', () => {
			expect(tabBar).toContain('tabs.tabs');
			expect(tabBar).toContain('tabs.activeTabId');
		});

		it('active tab has primary green bottom border and semibold text', () => {
			expect(tabBar).toContain('border-b-2');
			expect(tabBar).toContain('border-primary');
			expect(tabBar).toContain('font-semibold');
		});

		it('inactive tabs have muted text and no border', () => {
			expect(tabBar).toContain('text-on-surface-variant');
			const activePart = tabBar.match(/tab\.id === activeId[^{]*{([^}]*)}/s)?.[0] ?? '';
			// Inactive state is the else branch — confirms it lacks border classes
			expect(tabBar).toContain('hover:text-on-surface');
		});

		it('close button appears on hover for non-pinned tabs', () => {
			expect(tabBar).toContain('opacity-0');
			expect(tabBar).toContain('group-hover:opacity-100');
		});

		it('pinned tab has no close button', () => {
			expect(tabBar).toContain('!tab.pinned');
		});

		it('clicking a tab switches to it', () => {
			expect(tabBar).toContain('tabs.switchTab');
		});

		it('clicking close removes the tab', () => {
			expect(tabBar).toContain('tabs.closeTab');
		});

		it('plus button opens dropdown menu', () => {
			expect(tabBar).toContain('aria-label="Open new tab"');
			expect(tabBar).toContain('tabs.getAvailablePages');
		});

		it('dropdown marks open tabs with checkmark', () => {
			expect(tabBar).toContain('tabs.isTabOpen(page.id)');
		});

		it('clicking a page in dropdown calls openTab', () => {
			expect(tabBar).toContain('tabs.openTab(id)');
		});
	});

	describe('mobile bottom nav', () => {
		it('has Inbox, Customers, Dashboards, Copilots, Settings tabs', () => {
			expect(layout).toContain("label: 'Inbox'");
			expect(layout).toContain("label: 'Customers'");
			expect(layout).toContain("label: 'Dashboards'");
			expect(layout).toContain("label: 'Copilots'");
			expect(layout).toContain("label: 'Settings'");
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
