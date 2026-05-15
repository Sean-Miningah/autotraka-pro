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
	class="w-full border-2 border-text bg-base px-3 py-2 font-body text-text shadow-[4px_4px_0px] shadow-text placeholder:text-text/50 focus:translate-x-[2px] focus:translate-y-[2px] focus:shadow-[2px_2px_0px] focus:outline-none disabled:opacity-50 dark:border-text-dark dark:bg-base-dark dark:text-text-dark dark:shadow-text-dark dark:placeholder:text-text-dark/50 {className}"
/>