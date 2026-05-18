<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { ws } from '$lib/stores/websocket';
	import { auth } from '$lib/stores/auth';
	import { tabs } from '$lib/stores/tabs';
	import GlobalTopBar from '$lib/ui/GlobalTopBar.svelte';
	import DesktopTabBar from '$lib/ui/DesktopTabBar.svelte';

	interface LayoutData {
		accessToken: string | null;
	}

	let { children, data }: { children: import('svelte').Snippet; data: LayoutData } = $props();

	let currentPath = $derived($page.url.pathname);

	const mobileTabs = [
		{ id: 'inbox', label: 'Inbox', href: '/inbox' },
		{ id: 'customers', label: 'Customers', href: '/customers' },
		{ id: 'dashboards', label: 'Dashboards', href: '/dashboards' },
		{ id: 'copilots', label: 'Copilots', href: '/copilots' },
		{ id: 'settings', label: 'Settings', href: '/settings' }
	];

	function isActive(href: string): boolean {
		if (href === '/dashboards') return currentPath === '/' || currentPath.startsWith('/dashboards');
		return currentPath.startsWith(href);
	}

	$effect(() => {
		const token = data.accessToken;
		if (token) {
			auth.setToken(token);
		}
		ws.connect();

		return () => {
			ws.disconnect();
		};
	});

	// URL → tab: when the URL changes, sync tab state from the URL
	$effect(() => {
		const path = currentPath;
		tabs.syncFromUrl(path);
	});

	// Tab → URL: when the active tab changes programatically, navigate to its href
	// Guard: only navigate if the URL does not already "belong" to this tab
	// (preserves sub-routes like /inbox/abc123)
	$effect(() => {
		const activeId = tabs.activeTabId;
		const href = tabs.redirectToActiveTab();
		if (!currentPath.startsWith(href)) {
			goto(href);
		}
	});

	// Sync browser tab title with active tab label
	$effect(() => {
		const label = tabs.getActiveTabLabel();
		document.title = `${label} — Autotraka`;
	});
</script>

<div class="min-h-screen bg-surface">
	<div class="hidden lg:flex lg:flex-col lg:h-screen">
		<GlobalTopBar />
		<DesktopTabBar />

		<div class="flex-1 overflow-auto">
			{@render children()}
		</div>
	</div>

	<div class="lg:hidden">
		<main class="pb-16">
			{@render children()}
		</main>

		<nav class="fixed inset-x-0 bottom-0 z-30 border-t border-outline-variant bg-surface-container">
			<div class="flex items-center justify-around">
				{#each mobileTabs as tab (tab.id)}
					<a
						href={tab.href}
						class="flex flex-col items-center gap-1 px-3 py-2 font-heading text-xs font-semibold transition-colors {isActive(tab.href)
							? 'text-primary'
							: 'text-on-surface-variant'}"
					>
						{tab.label}
					</a>
				{/each}
			</div>
		</nav>
	</div>
</div>