<script lang="ts">
	interface Props {
		src?: string;
		alt?: string;
		name?: string;
		class?: string;
	}

	let { src = '', alt = '', name = '', class: className = '' }: Props = $props();
	const initials = $derived(name ? name.split(' ').map((n) => n[0]).join('').toUpperCase().slice(0, 2) : '?');
</script>

{#if src}
	<img {src} {alt} class="h-10 w-10 rounded-full border-2 border-text object-cover {className}" />
{:else}
	<div class="flex h-10 w-10 items-center justify-center rounded-full border-2 border-text bg-surface font-heading font-bold text-text dark:border-text-dark dark:bg-surface-dark dark:text-text-dark {className}">
		{initials}
	</div>
{/if}