import { describe, it, expect } from 'vitest';
import { get } from 'svelte/store';
import { createTabStore, type Tab } from './tabs';

describe('tab store', () => {
	it('starts with Dashboards pinned and active by default', () => {
		const store = createTabStore();
		const tabs = get(store);
		
		expect(tabs.length).toBe(1);
		expect(tabs[0].id).toBe('dashboards');
		expect(tabs[0].pinned).toBe(true);
		expect(store.activeTabId).toBe('dashboards');
	});

	it('openTab creates a new tab and makes it active', () => {
		const store = createTabStore();
		store.openTab('inbox');

		const tabs = get(store);
		expect(tabs.length).toBe(2);
		expect(tabs[1].id).toBe('inbox');
		expect(tabs[1].pinned).toBe(false);
		expect(store.activeTabId).toBe('inbox');
	});

	it('openTab on existing tab switches to it without duplicating', () => {
		const store = createTabStore();
		store.openTab('inbox');
		store.openTab('customers');
		// inbox is active, customers is also open

		store.openTab('inbox');

		const tabs = get(store);
		expect(tabs.length).toBe(3); // dashboards, inbox, customers (no duplicate)
		expect(store.activeTabId).toBe('inbox');
	});

	it('switchTab activates an existing tab', () => {
		const store = createTabStore();
		store.openTab('inbox');
		store.openTab('analytics');
		// analytics is now active

		store.switchTab('inbox');

		expect(store.activeTabId).toBe('inbox');
	});

	it('switchTab is a no-op for tabs that are not open', () => {
		const store = createTabStore();
		store.openTab('inbox');
		// dashboards and inbox are open; analytics is not

		store.switchTab('analytics');

		expect(store.activeTabId).toBe('inbox');
	});

	it('closeTab on active tab removes it and activates the left neighbor', () => {
		const store = createTabStore();
		store.openTab('inbox');   // position 1
		store.openTab('customers'); // position 2, active
		// tabs: dashboards[0], inbox[1], customers[2]

		store.closeTab('customers'); // close active tab, left neighbor is inbox

		const tabs = get(store);
		expect(tabs.map((t) => t.id)).toEqual(['dashboards', 'inbox']);
		expect(store.activeTabId).toBe('inbox');
	});

	it('closeTab on active middle tab activates the left neighbor', () => {
		const store = createTabStore();
		store.openTab('inbox');     // position 1
		store.openTab('customers'); // position 2
		store.openTab('analytics'); // position 3
		store.switchTab('customers'); // explicitly make customers active
		// tabs: dashboards[0], inbox[1], customers[2], analytics[3]

		store.closeTab('customers'); // close active tab, left neighbor is inbox

		const tabs = get(store);
		expect(tabs.map((t) => t.id)).toEqual([
			'dashboards', 'inbox', 'analytics'
		]);
		expect(store.activeTabId).toBe('inbox');
	});

	it('closeTab on rightmost active tab falls back to the remaining rightmost', () => {
		const store = createTabStore();
		store.openTab('inbox');     // position 1, active
		// tabs: dashboards[0], inbox[1]

		store.closeTab('inbox'); // no left neighbor, fallback to dashboards

		const tabs = get(store);
		expect(tabs.map((t) => t.id)).toEqual(['dashboards']);
		expect(store.activeTabId).toBe('dashboards');
	});

	it('closeTab on non-active tab does not change the active tab', () => {
		const store = createTabStore();
		store.openTab('inbox'); // active
		store.openTab('customers'); // not active (if active is inbox)
		// Actually let me re-read: inbox is active. Then customers is opened and becomes active.
		// So after openTab('customers'), active is customers. Let me just switch back to inbox.
		store.switchTab('inbox');
		// tabs: dashboards[0], inbox[1], customers[2]. Active: inbox[1]

		store.closeTab('customers'); // close non-active tab

		const tabs = get(store);
		expect(tabs.map((t) => t.id)).toEqual(['dashboards', 'inbox']);
		expect(store.activeTabId).toBe('inbox'); // does not switch
	});

	it('closeTab does not remove the pinned Dashboards tab', () => {
		const store = createTabStore();
		store.closeTab('dashboards');

		const tabs = get(store);
		expect(tabs.length).toBe(1);
		expect(tabs[0].id).toBe('dashboards');
		expect(tabs[0].pinned).toBe(true);
	});

	it('isTabOpen returns true for open tabs and false otherwise', () => {
		const store = createTabStore();
		expect(store.isTabOpen('dashboards')).toBe(true);
		expect(store.isTabOpen('inbox')).toBe(false);

		store.openTab('inbox');
		expect(store.isTabOpen('inbox')).toBe(true);
		expect(store.isTabOpen('analytics')).toBe(false);
	});

	it('getAvailablePages returns all page types for the plus dropdown', () => {
		const store = createTabStore();
		const pages = store.getAvailablePages();

		expect(pages.length).toBe(6);
		expect(pages.map((p) => p.id)).toEqual([
			'dashboards', 'inbox', 'customers', 'analytics', 'copilots', 'settings'
		]);
	});

	it('tab order follows open order with Dashboards always leftmost', () => {
		const store = createTabStore();
		store.openTab('customers');
		store.openTab('analytics');
		store.openTab('copilots');
		// Expected order: dashboards[0], customers[1], analytics[2], copilots[3]

		const tabs = get(store);
		expect(tabs.map((t) => t.id)).toEqual([
			'dashboards', 'customers', 'analytics', 'copilots'
		]);
		expect(tabs[0].pinned).toBe(true);
	});

	it('syncFromUrl opens the corresponding tab and makes it active', () => {
		const store = createTabStore();
		store.syncFromUrl('/customers');

		const tabs = get(store);
		expect(tabs.length).toBe(2);
		expect(tabs[1].id).toBe('customers');
		expect(tabs[1].pinned).toBe(false);
		expect(store.activeTabId).toBe('customers');
	});

	it('syncFromUrl maps sub-routes to the parent tab', () => {
		const store = createTabStore();
		store.syncFromUrl('/inbox/abc123');

		const tabs = get(store);
		expect(tabs.length).toBe(2);
		expect(tabs[1].id).toBe('inbox');
		expect(tabs[1].href).toBe('/inbox');
		expect(store.activeTabId).toBe('inbox');
	});

	it('syncFromUrl defaults to dashboards for unknown paths', () => {
		const store = createTabStore();
		store.syncFromUrl('/unknown');

		const tabs = get(store);
		expect(tabs.length).toBe(1); // only dashboards
		expect(tabs[0].id).toBe('dashboards');
		expect(store.activeTabId).toBe('dashboards');
	});

	it('syncFromUrl switches to already-open tab without duplicating', () => {
		const store = createTabStore();
		store.openTab('inbox');
		store.openTab('customers');
		// Inbox and customers are already open; active is customers

		store.syncFromUrl('/inbox');

		const tabs = get(store);
		expect(tabs.length).toBe(3); // dashboards, inbox, customers (no duplicate)
		expect(store.activeTabId).toBe('inbox');
	});

	it('redirectToActiveTab returns the href of the active tab', () => {
		const store = createTabStore();
		store.openTab('inbox');
		// activeTabId = 'inbox'
		expect(store.redirectToActiveTab()).toBe('/inbox');
	});

	it('redirectToActiveTab returns /dashboards when no matching tab is found', () => {
		const store = createTabStore();
		// dashboards is the only tab active by default
		expect(store.redirectToActiveTab()).toBe('/dashboards');
	});
});
