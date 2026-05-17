<script lang="ts">
	import { tabs } from '$lib/stores/tabs';

	let activeId = $derived(tabs.activeTabId);
	let tabList = $derived(tabs.tabs);
	let menuOpen = $state(false);

	function handleTabClick(id: string) {
		tabs.switchTab(id);
	}

	function handleClose(e: MouseEvent, id: string) {
		e.stopPropagation();
		tabs.closeTab(id);
	}

	function handleOpenPage(id: string) {
		tabs.openTab(id);
		menuOpen = false;
	}

	function handlePlusClick() {
		menuOpen = !menuOpen;
	}

	function handleClickOutside() {
		menuOpen = false;
	}
</script>

<div class="relative flex items-center border-b border-outline-variant bg-surface" >
	{#each tabList as tab (tab.id)}
		<button
			class="group relative flex h-9 items-center gap-1.5 px-3 font-heading text-sm transition-colors {tab.id === activeId
				? 'text-on-surface border-b-2 border-primary font-semibold'
				: 'text-on-surface-variant hover:text-on-surface hover:bg-surface-container'}"
			onclick={() => handleTabClick(tab.id)}
		>
			{tab.label}
			{#if !tab.pinned}
				<button
					class="ml-1 flex h-4 w-4 items-center justify-center rounded-[var(--radius-sm)] text-on-surface-variant hover:bg-surface-container-high hover:text-on-surface opacity-0 group-hover:opacity-100 transition-opacity"
					onclick={(e) => handleClose(e, tab.id)}
					aria-label="Close {tab.label}"
				>
					<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
				</button>
			{/if}
		</button>
	{/each}

	<div class="relative">
		<button
			class="flex h-9 w-9 items-center justify-center text-on-surface-variant hover:text-on-surface hover:bg-surface-container transition-colors"
			onclick={handlePlusClick}
			aria-label="Open new tab"
		>
			<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 5v14"/><path d="M5 12h14"/></svg>
		</button>

		{#if menuOpen}
			<div class="fixed inset-0 z-40" onclick={handleClickOutside} onkeydown={() => {}} role="presentation"></div>
			<div class="absolute left-0 top-full z-50 mt-1 w-48 rounded-[var(--radius-default)] border border-outline-variant bg-surface-container-lowest shadow-[var(--shadow-elevation-2)] py-1">
				{#each tabs.getAvailablePages() as page (page.id)}
					<button
						class="flex w-full items-center gap-2 px-3 py-2 text-sm transition-colors {page.id === activeId || tabs.isTabOpen(page.id)
							? 'text-primary font-semibold bg-surface-container'
							: 'text-on-surface hover:bg-surface-container-high'}"
						onclick={() => handleOpenPage(page.id)}
					>
						{page.label}
						{#if tabs.isTabOpen(page.id)}
							<svg class="ml-auto" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5"/></svg>
						{/if}
					</button>
				{/each}
			</div>
		{/if}
	</div>
</div>
