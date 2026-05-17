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

<div class="flex border-b border-outline-variant {className}">
	{#each tabs as tab (tab.id)}
		<button
			class="px-4 py-2 font-heading text-sm font-semibold border-b-2 transition-colors {tab.id === activeId
				? 'border-primary text-on-surface'
				: 'border-transparent text-on-surface/50 hover:text-on-surface hover:border-outline-variant'}"
			onclick={() => onselect(tab.id)}
		>
			{tab.label}
		</button>
	{/each}
</div>