import { writable, get } from 'svelte/store';

export interface PageType {
	id: string;
	label: string;
	href: string;
	icon?: string;
}

export interface Tab extends PageType {
	pinned: boolean;
}

const PAGE_TYPES: PageType[] = [
	{ id: 'dashboards', label: 'Dashboards', href: '/dashboards', icon: 'dashboard' },
	{ id: 'inbox', label: 'Inbox', href: '/inbox', icon: 'inbox' },
	{ id: 'customers', label: 'Customers', href: '/customers', icon: 'customers' },
	{ id: 'analytics', label: 'Analytics', href: '/analytics', icon: 'analytics' },
	{ id: 'copilots', label: 'Copilots', href: '/copilots', icon: 'copilots' },
	{ id: 'settings', label: 'Settings', href: '/settings', icon: 'settings' }
];

const PINNED_TAB: Tab = {
	...PAGE_TYPES[0],
	pinned: true
};

export const pageTypes = PAGE_TYPES;

export function createTabStore() {
	const store = writable<Tab[]>([PINNED_TAB]);
	const activeId = writable<string>(PINNED_TAB.id);

	function doOpenTab(pageId: string) {
		const $tabs = get(store);
		const existing = $tabs.find((t) => t.id === pageId);
		if (existing) {
			activeId.set(pageId);
			return;
		}
		const pageType = PAGE_TYPES.find((p) => p.id === pageId);
		if (!pageType) return;
		const newTab: Tab = { ...pageType, pinned: false };
		store.update((t) => [...t, newTab]);
		activeId.set(pageId);
	}

	return {
		subscribe: store.subscribe,
		subscribeActiveId: activeId.subscribe,

		get tabs(): Tab[] {
			return get(store);
		},

		get activeTabId(): string {
			return get(activeId);
		},

		openTab(pageId: string) {
			doOpenTab(pageId);
		},

		switchTab(pageId: string) {
			const $tabs = get(store);
			if ($tabs.find((t) => t.id === pageId)) {
				activeId.set(pageId);
			}
		},

		closeTab(pageId: string) {
			const $tabs = get(store);
			const tab = $tabs.find((t) => t.id === pageId);
			if (!tab || tab.pinned) return;

			const $activeId = get(activeId);
			const index = $tabs.indexOf(tab);
			const remaining = $tabs.filter((t) => t.id !== pageId);
			store.set(remaining);

			if ($activeId === pageId) {
				const neighborIndex = index - 1 >= 0 ? index - 1 : remaining.length - 1;
				const neighbor = remaining[neighborIndex];
				if (neighbor) {
					activeId.set(neighbor.id);
				}
			}
		},

		isTabOpen(pageId: string): boolean {
			return get(store).some((t) => t.id === pageId);
		},

		getAvailablePages(): PageType[] {
			const $tabs = get(store);
			return PAGE_TYPES;
		},

		syncFromUrl(pathname: string) {
			const parts = pathname.split('/').filter(Boolean);
			const rootPath = parts[0] ? '/' + parts[0] : '/';
			const matched = PAGE_TYPES.find((p) => rootPath === p.href);
			doOpenTab(matched ? matched.id : 'dashboards');
		},

		redirectToActiveTab(): string {
			const id = get(activeId);
			const pageType = PAGE_TYPES.find((p) => p.id === id);
			return pageType ? pageType.href : '/dashboards';
		}
	};
}

export const tabs = createTabStore();