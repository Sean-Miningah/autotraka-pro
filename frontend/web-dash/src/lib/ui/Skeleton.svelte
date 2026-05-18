<script lang="ts">
	interface Props {
		class?: string;
		height?: string;
		width?: string;
	}
	let { class: className = '', height = 'h-32', width = 'w-full' }: Props = $props();
</script>

<div class="animate-pulse bg-surface-container rounded-[var(--radius-default)] {height} {width} {className}"></div>
