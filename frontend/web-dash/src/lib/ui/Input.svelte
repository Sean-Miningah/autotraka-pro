<script lang="ts">
	interface Props {
		type?: 'text' | 'password' | 'email' | 'search' | 'number';
		value?: string;
		placeholder?: string;
		disabled?: boolean;
		name?: string;
		required?: boolean;
		class?: string;
		oninput?: (e: Event) => void;
		onchange?: (e: Event) => void;
	}

	let {
		type = 'text',
		value = $bindable(''),
		placeholder = '',
		disabled = false,
		name = '',
		required = false,
		class: className = '',
		oninput,
		onchange
	}: Props = $props();
</script>

<input
	{type}
	bind:value
	{name}
	{placeholder}
	{disabled}
	{required}
	{oninput}
	{onchange}
	class="w-full border border-outline-variant bg-surface-container-lowest px-3 py-2 font-body text-on-surface rounded-[var(--radius-default)] placeholder:text-on-surface/40 focus:border-primary focus:ring-2 focus:ring-primary/20 focus:outline-none transition-colors disabled:opacity-50 {className}"
/>