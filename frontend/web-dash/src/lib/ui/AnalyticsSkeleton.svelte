<div class="mx-auto max-w-4xl animate-pulse space-y-6 p-8">
	<!-- Metric cards -->
	<div class="grid grid-cols-1 gap-4 md:grid-cols-3">
		{#each [1, 2, 3] as _}
			<div class="space-y-3 rounded-[var(--radius-default)] bg-surface-container p-4">
				<div class="h-4 w-3/4 rounded bg-surface-container-high"></div>
				<div class="h-8 w-1/2 rounded bg-surface-container-high"></div>
			</div>
		{/each}
	</div>

	<!-- Chart areas -->
	<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
		<div class="h-48 rounded-[var(--radius-default)] bg-surface-container"></div>
		<div class="h-48 rounded-[var(--radius-default)] bg-surface-container"></div>
	</div>
</div>
