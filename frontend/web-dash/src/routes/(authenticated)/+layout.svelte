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
		{ id: 'contacts', label: 'Contacts', href: '/contacts' },
		{ id: 'mystats', label: 'My Stats', href: '/analytics' },
		{ id: 'profile', label: 'Profile', href: '/settings' }
	];

	const desktopNavItems = [
		{ id: 'inbox', label: 'Inbox', href: '/inbox' },
		{ id: 'contacts', label: 'Contacts', href: '/contacts' },
		{ id: 'analytics', label: 'Analytics', href: '/analytics' },
		{ id: 'templates', label: 'Templates', href: '/templates' },
		{ id: 'settings', label: 'Settings', href: '/settings' }
	];

	function isActive(href: string): boolean {
		if (href === '/inbox') return currentPath === '/' || currentPath.startsWith('/inbox');
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

<div class="min-h-screen bg-base dark:bg-base-dark">
	<!-- Desktop sidebar (hidden on mobile) -->
	<aside class="hidden lg:fixed lg:inset-y-0 lg:left-0 lg:z-30 lg:flex lg:w-64 lg:flex-col lg:border-r-2 lg:border-text lg:bg-surface dark:lg:border-text-dark dark:lg:bg-surface-dark">
		<div class="flex h-16 items-center border-b-2 border-text px-6 dark:border-text-dark">
			<h1 class="font-heading text-xl font-bold text-text dark:text-text-dark">
				<span class="text-primary">Auto</span>traka
			</h1>
		</div>
		<nav class="flex-1 space-y-1 px-3 py-4">
			{#each desktopNavItems as item (item.id)}
				<a
					href={item.href}
					class="flex items-center gap-3 border-2 px-3 py-2.5 font-heading text-sm font-semibold transition-all {isActive(item.href)
						? 'border-text bg-primary text-text shadow-[4px_4px_0px] shadow-text dark:border-text-dark dark:shadow-text-dark'
						: 'border-transparent text-text/70 hover:border-text hover:bg-surface hover:text-text dark:text-text-dark/70 dark:hover:border-text-dark dark:hover:bg-surface-dark dark:hover:text-text-dark'}"
				>
					{item.label}
				</a>
			{/each}
		</nav>
	</aside>

	<!-- Main content area (with sidebar offset on desktop) -->
	<main class="lg:pl-64">
		{@render children()}
	</main>

	<!-- Mobile bottom tab bar (hidden on desktop) -->
	<nav class="fixed inset-x-0 bottom-0 z-30 border-t-2 border-text bg-surface lg:hidden dark:border-text-dark dark:bg-surface-dark">
		<div class="flex items-center justify-around">
			{#each mobileTabs as tab (tab.id)}
				<a
					href={tab.href}
					class="flex flex-col items-center gap-1 px-3 py-2 font-heading text-xs font-semibold {isActive(tab.href)
						? 'text-primary'
						: 'text-text/60 dark:text-text-dark/60'}"
				>
					{tab.label}
				</a>
			{/each}
		</div>
	</nav>
</div>