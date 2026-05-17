<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		open: boolean;
		onclose: () => void;
		title?: string;
		class?: string;
		children: Snippet;
	}

	let { open, onclose, title = '', class: className = '', children }: Props = $props();
</script>

{#if open}
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-on-surface/30" onclick={onclose} onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onclose(); } }} role="button" tabindex={-1}></div>
	<div class="fixed inset-0 z-50 flex items-center justify-center pointer-events-none">
		<div class="pointer-events-auto w-full max-w-lg bg-surface-container-lowest border border-outline-variant rounded-[var(--radius-default)] p-6 shadow-[var(--shadow-elevation-2)] {className}" onclick={(e) => e.stopPropagation()} onkeydown={(e) => { if (e.key === 'Escape') onclose(); }} role="dialog" aria-modal="true" tabindex={-1}>
			{#if title}
				<h2 class="mb-4 font-heading text-xl font-bold text-on-surface">{title}</h2>
			{/if}
			{@render children()}
		</div>
	</div>
{/if}