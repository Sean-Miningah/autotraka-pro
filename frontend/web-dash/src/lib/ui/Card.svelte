<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		class?: string;
		children: Snippet;
	}

	let { class: className = '', children }: Props = $props();
</script>

<div class="border-2 border-text bg-surface p-4 shadow-[4px_4px_0px] shadow-text dark:border-text-dark dark:bg-surface-dark dark:shadow-text-dark {className}">
	{@render children()}
</div>