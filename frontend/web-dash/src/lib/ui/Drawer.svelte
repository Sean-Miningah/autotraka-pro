<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		open: boolean;
		onclose: () => void;
		side?: 'left' | 'right' | 'bottom';
		class?: string;
		children: Snippet;
	}

	let { open, onclose, side = 'right', class: className = '', children }: Props = $props();

	const sideClasses = {
		right: 'fixed inset-y-0 right-0 w-full max-w-md translate-x-0',
		left: 'fixed inset-y-0 left-0 w-full max-w-md translate-x-0',
		bottom: 'fixed inset-x-0 bottom-0 h-auto max-h-[80vh] translate-y-0'
	};
</script>

{#if open}
	<div class="fixed inset-0 z-40 bg-on-surface/30" onclick={onclose} onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onclose(); } }} role="button" tabindex={-1} aria-label="Close drawer"></div>
	<div class="z-50 bg-surface-container-lowest border-l border-outline-variant p-4 shadow-[var(--shadow-elevation-2)] {sideClasses[side]} {className}">
		<button class="absolute right-2 top-2 text-on-surface hover:text-error transition-colors" onclick={onclose} aria-label="Close">
			<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
		</button>
		{@render children()}
	</div>
{/if}