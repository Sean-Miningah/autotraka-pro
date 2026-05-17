<script lang="ts">
	import { tabs } from '$lib/stores/tabs';

	interface Props {
		pageId: string;
		children: import('svelte').Snippet;
	}

	let { pageId, children }: Props = $props();

	let isActive = $derived(tabs.activeTabId === pageId);
</script>

<div class="h-full" style:display={isActive ? 'contents' : 'none'}>
	{@render children()}
</div>