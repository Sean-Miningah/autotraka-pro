<script lang="ts">
	type Tab = { id: string; label: string };

	interface Props {
		tabs: Tab[];
		activeId: string;
		onselect: (id: string) => void;
		class?: string;
	}

	let { tabs, activeId, onselect, class: className = '' }: Props = $props();
</script>

<div class="flex border-b-2 border-text dark:border-text-dark {className}">
	{#each tabs as tab (tab.id)}
		<button
			class="px-4 py-2 font-heading text-sm font-semibold border-b-2 transition-colors {tab.id === activeId
				? 'border-primary text-text dark:text-text-dark bg-primary/10'
				: 'border-transparent text-text/60 dark:text-text-dark/60 hover:text-text dark:hover:text-text-dark'}"
			onclick={() => onselect(tab.id)}
		>
			{tab.label}
		</button>
	{/each}
</div>