<script lang="ts">
	type BadgeVariant = 'default' | 'primary' | 'secondary' | 'success' | 'danger' | 'whatsapp' | 'instagram' | 'facebook';

	interface Props {
		variant?: BadgeVariant;
		class?: string;
	}

	let { variant = 'default', class: className = '' }: Props = $props();

	const variantClasses: Record<BadgeVariant, string> = {
		default: 'bg-surface text-text border-text dark:bg-surface-dark dark:text-text-dark dark:border-text-dark',
		primary: 'bg-primary text-text border-text',
		secondary: 'bg-secondary text-text border-text',
		success: 'bg-success text-text border-text',
		danger: 'bg-danger text-white border-text',
		whatsapp: 'bg-whatsapp text-text border-text',
		instagram: 'bg-instagram text-white border-text',
		facebook: 'bg-facebook text-white border-text'
	};
</script>

<span class="inline-flex items-center rounded-sm border-2 px-2 py-0.5 text-xs font-heading font-semibold {variantClasses[variant]} {className}">
	<slot />
</span>