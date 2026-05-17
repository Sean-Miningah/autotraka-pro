<script lang="ts">
	type BadgeVariant = 'default' | 'primary' | 'secondary' | 'success' | 'danger' | 'tonal' | 'whatsapp' | 'instagram' | 'facebook';

	interface Props {
		variant?: BadgeVariant;
		class?: string;
		children?: import('svelte').Snippet;
	}

	let { variant = 'default', class: className = '', children }: Props = $props();

	const variantClasses: Record<BadgeVariant, string> = {
		default: 'bg-on-surface/5 text-on-surface',
		primary: 'bg-primary/10 text-on-primary-container',
		secondary: 'bg-secondary/10 text-on-secondary-container',
		success: 'bg-primary/10 text-on-primary-container',
		danger: 'bg-error/10 text-on-error-container',
		tonal: 'bg-on-surface/8 text-on-surface',
		whatsapp: 'bg-whatsapp/10 text-[#005523]',
		instagram: 'bg-instagram/10 text-[#a02b47]',
		facebook: 'bg-facebook/10 text-[#0e4ea3]'
	};
</script>

<span class="inline-flex items-center rounded-[var(--radius-full)] px-2.5 py-0.5 text-xs font-heading font-semibold {variantClasses[variant]} {className}">
	{#if children}{@render children()}{/if}
</span>