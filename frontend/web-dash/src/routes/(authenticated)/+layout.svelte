<script lang="ts">
	import { page } from '$app/stores';
	import { ws } from '$lib/stores/websocket';
	import { auth } from '$lib/stores/auth';

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

	const desktopNavItems = [
		{ id: 'dashboards', label: 'Dashboards', href: '/dashboards' },
		{ id: 'inbox', label: 'Inbox', href: '/inbox' },
		{ id: 'customers', label: 'Customers', href: '/customers' },
		{ id: 'analytics', label: 'Analytics', href: '/analytics' },
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
</script>

<div class="min-h-screen bg-surface">
	<aside class="hidden lg:fixed lg:inset-y-0 lg:left-0 lg:z-30 lg:flex lg:w-64 lg:flex-col lg:border-r lg:border-outline-variant lg:bg-surface-container">
		<div class="flex h-16 items-center border-b border-outline-variant px-6">
			<h1 class="font-heading text-xl font-bold text-on-surface">
				<span class="text-primary">Auto</span>traka
			</h1>
		</div>
		<nav class="flex-1 space-y-1 px-3 py-4">
			{#each desktopNavItems as item (item.id)}
				<a
					href={item.href}
					class="flex items-center gap-3 rounded-[var(--radius-default)] px-3 py-2.5 font-heading text-sm font-semibold transition-colors {isActive(item.href)
						? 'bg-primary-container text-on-primary-container border-l-[4px] border-primary'
						: 'text-on-surface-variant hover:bg-surface-container-high'}"
				>
					{item.label}
				</a>
			{/each}
		</nav>
	</aside>

	<main class="lg:pl-64">
		{@render children()}
	</main>

	<nav class="fixed inset-x-0 bottom-0 z-30 border-t border-outline-variant bg-surface-container lg:hidden">
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