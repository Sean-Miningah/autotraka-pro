<script lang="ts">
	let { children } = $props();
</script>

<div class="min-h-screen bg-surface">
	{@render children()}
</div>