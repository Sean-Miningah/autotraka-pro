<script lang="ts">
	import type { Snippet } from 'svelte';

	type ButtonVariant = 'primary' | 'secondary' | 'danger' | 'success' | 'ghost';
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
		primary: 'bg-primary text-text hover:bg-primary/90 border-text',
		secondary: 'bg-secondary text-text hover:bg-secondary/90 border-text',
		danger: 'bg-danger text-white hover:bg-danger/90 border-text',
		success: 'bg-success text-text hover:bg-success/90 border-text',
		ghost: 'bg-transparent text-text hover:bg-surface border-text'
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
	class="inline-flex items-center justify-center font-heading font-semibold border-2 shadow-[4px_4px_0px_var(--tw-shadow-color)] shadow-text transition-all active:translate-x-[2px] active:translate-y-[2px] active:shadow-[2px_2px_0px_var(--tw-shadow-color)] disabled:opacity-50 disabled:cursor-not-allowed {variantClasses[variant]} {sizeClasses[size]} {className}"
>
	{@render children()}
</button>