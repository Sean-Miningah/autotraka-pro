<div class="mx-auto max-w-4xl animate-pulse space-y-6 p-8">
	<!-- 3 metric cards -->
	<div class="grid grid-cols-3 gap-4">
		{#each [1, 2, 3] as _}
			<div class="space-y-3 rounded-[var(--radius-default)] bg-surface-container p-4">
				<div class="h-4 w-3/4 rounded bg-surface-container-high"></div>
				<div class="h-8 w-1/2 rounded bg-surface-container-high"></div>
			</div>
		{/each}
	</div>

	<!-- Chart area -->
	<div class="h-64 rounded-[var(--radius-default)] bg-surface-container"></div>
</div>
