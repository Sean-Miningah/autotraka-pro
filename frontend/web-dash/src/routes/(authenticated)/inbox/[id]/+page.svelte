<script lang="ts">
	import { Badge, Button, ChannelBadge, StatusPill } from '$lib/ui';

	let showAiSummary = $state(false);
</script>

<div class="flex h-full flex-col bg-surface">
	<!-- Conversation header -->
	<div class="flex items-center justify-between border-b border-outline-variant px-4 py-3">
		<h1 class="font-heading text-[28px] font-bold leading-tight text-on-surface">Conversation</h1>
		<div class="flex items-center gap-2">
			<Button variant="outline" size="sm" onclick={() => console.log('Voice call stub')}>
				<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z"/></svg>
			</Button>
		</div>
	</div>

	<!-- AI Summary panel -->
	<div class="border-b border-outline-variant">
		<button
			class="flex w-full items-center justify-between px-4 py-2 text-left font-heading text-sm font-semibold text-on-surface-variant hover:bg-surface-container-high transition-colors"
			onclick={() => showAiSummary = !showAiSummary}
		>
			AI Summary
			<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="transition-transform {showAiSummary ? 'rotate-180' : ''}"><polyline points="6 9 12 15 18 9"/></svg>
		</button>
		{#if showAiSummary}
			<div class="bg-surface-container px-4 py-3">
				<p class="text-on-surface-variant">AI-generated summary will appear here.</p>
			</div>
		{/if}
	</div>

	<!-- Placeholder content -->
	<div class="flex flex-1 items-center justify-center p-8">
		<p class="text-on-surface-variant">Thread view will appear here.</p>
	</div>
</div>