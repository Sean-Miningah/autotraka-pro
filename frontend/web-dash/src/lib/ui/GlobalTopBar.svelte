<script lang="ts">
	import { auth } from '$lib/stores/auth';
	import { tabs } from '$lib/stores/tabs';

	interface Props {
		class?: string;
	}

	let { class: className = '' }: Props = $props();

	function goDashboards() {
		tabs.openTab('dashboards');
	}
</script>

<header class="flex h-12 items-center justify-between border-b border-outline-variant bg-surface-container px-4 {className}">
	<button onclick={goDashboards} class="flex items-center gap-2 font-heading text-lg font-bold text-on-surface hover:opacity-80 transition-opacity">
		<span class="text-primary">Auto</span><span>traka</span>
	</button>

	<div class="flex items-center gap-3">
		<button
			class="flex h-8 w-8 items-center justify-center rounded-[var(--radius-default)] text-on-surface-variant hover:bg-surface-container-high transition-colors"
			aria-label="Search"
		>
			<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
		</button>

		<button
			class="flex h-8 w-8 items-center justify-center rounded-[var(--radius-default)] text-on-surface-variant hover:bg-surface-container-high transition-colors"
			aria-label="Notifications"
		>
			<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9"/><path d="M10.3 21a1.94 1.94 0 0 0 3.4 0"/></svg>
		</button>

		<button
			class="flex h-8 w-8 items-center justify-center rounded-[var(--radius-full)] bg-primary text-on-primary font-heading text-xs font-semibold hover:opacity-90 transition-opacity"
			aria-label="User menu"
		>
			U
		</button>
	</div>
</header>