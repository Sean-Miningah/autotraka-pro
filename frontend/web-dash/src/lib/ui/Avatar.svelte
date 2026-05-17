<script lang="ts">
	interface Props {
		src?: string;
		alt?: string;
		fallback?: string;
		size?: 'sm' | 'md' | 'lg';
		class?: string;
	}

	let { src = '', alt = '', fallback = '', size = 'md', class: className = '' }: Props = $props();

	const sizeClasses: Record<string, string> = {
		sm: 'w-8 h-8 text-xs',
		md: 'w-10 h-10 text-sm',
		lg: 'w-12 h-12 text-base'
	};

	const initials = $derived.by(() => {
		if (!fallback && !alt) return '?';
		const name = fallback || alt;
		return name.split(' ').map(w => w[0]).join('').toUpperCase().slice(0, 2);
	});
</script>

{#if src}
	<img {src} {alt} class="ring-1 ring-outline-variant rounded-[var(--radius-full)] object-cover {sizeClasses[size]} {className}" />
{:else}
	<div class="ring-1 ring-outline-variant rounded-[var(--radius-full)] bg-primary/10 text-on-primary-container flex items-center justify-center font-heading font-semibold {sizeClasses[size]} {className}">
		{initials}
	</div>
{/if}