<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		class?: string;
		children: Snippet;
	}

	let { class: className = '', children }: Props = $props();
</script>

<div class="bg-surface-container-lowest border border-outline-variant rounded-[var(--radius-default)] p-4 shadow-[var(--shadow-elevation-1)] {className}">
	{@render children()}
</div>