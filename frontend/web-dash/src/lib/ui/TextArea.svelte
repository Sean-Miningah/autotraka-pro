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
	class="w-full border border-outline-variant bg-surface-container-lowest px-3 py-2 font-body text-on-surface rounded-[var(--radius-default)] placeholder:text-on-surface/40 focus:border-primary focus:ring-2 focus:ring-primary/20 focus:outline-none transition-colors disabled:opacity-50 {className}"
></textarea>