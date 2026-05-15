<script lang="ts">
	interface Props {
		value?: string;
		placeholder?: string;
		rows?: number;
		disabled?: boolean;
		oninput?: (e: Event) => void;
		class?: string;
	}

	let { value = '', placeholder = '', rows = 3, disabled = false, oninput, class: className = '' }: Props = $props();
</script>

<textarea
	{value}
	{placeholder}
	{rows}
	{disabled}
	{oninput}
	class="w-full border-2 border-text bg-base px-3 py-2 font-body text-text shadow-[4px_4px_0px] shadow-text placeholder:text-text/50 focus:translate-x-[2px] focus:translate-y-[2px] focus:shadow-[2px_2px_0px] focus:outline-none disabled:opacity-50 dark:border-text-dark dark:bg-base-dark dark:text-text-dark dark:shadow-text-dark dark:placeholder:text-text-dark/50 {className}"
></textarea>