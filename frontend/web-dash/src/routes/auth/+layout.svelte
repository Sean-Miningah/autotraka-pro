<script lang="ts">
	let { children } = $props();
</script>

<div class="min-h-screen bg-base dark:bg-base-dark">
	{@render children()}
</div>