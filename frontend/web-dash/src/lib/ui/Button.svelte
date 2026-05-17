<script lang="ts">
	import type { Snippet } from 'svelte';

	type ButtonVariant = 'primary' | 'secondary' | 'danger' | 'success' | 'outline' | 'tonal';
	type ButtonSize = 'sm' | 'md' | 'lg';

	interface Props {
		variant?: ButtonVariant;
		size?: ButtonSize;
		disabled?: boolean;
		type?: 'button' | 'submit' | 'reset';
		onclick?: (e: MouseEvent) => void;
		children: Snippet;
		class?: string;
	}

	let {
		variant = 'primary',
		size = 'md',
		disabled = false,
		type = 'button',
		onclick,
		children,
		class: className = ''
	}: Props = $props();

	const variantClasses: Record<ButtonVariant, string> = {
		primary: 'bg-primary text-on-primary hover:bg-primary/90 active:bg-primary/80 focus-visible:ring-2 focus-visible:ring-primary/40',
		secondary: 'bg-secondary text-on-secondary hover:bg-secondary/90 active:bg-secondary/80 focus-visible:ring-2 focus-visible:ring-secondary/40',
		danger: 'bg-error text-on-error hover:bg-error/90 active:bg-error/80 focus-visible:ring-2 focus-visible:ring-error/40',
		success: 'bg-primary-container text-on-primary-container hover:bg-primary-container/90 active:bg-primary-container/80 focus-visible:ring-2 focus-visible:ring-primary-container/40',
		outline: 'bg-transparent text-on-surface border border-outline-variant hover:bg-surface-container-low active:bg-surface-container focus-visible:ring-2 focus-visible:ring-primary/40',
		tonal: 'bg-primary/10 text-on-primary-container hover:bg-primary/15 active:bg-primary/20 focus-visible:ring-2 focus-visible:ring-primary/40'
	};

	const sizeClasses: Record<ButtonSize, string> = {
		sm: 'px-3 py-1.5 text-sm',
		md: 'px-4 py-2 text-base',
		lg: 'px-6 py-3 text-lg'
	};
</script>

<button
	{type}
	{disabled}
	{onclick}
	class="inline-flex items-center justify-center font-heading font-semibold rounded-[var(--radius-default)] transition-colors focus-visible:outline-none disabled:opacity-50 disabled:cursor-not-allowed {variantClasses[variant]} {sizeClasses[size]} {className}"
>
	{@render children()}
</button>