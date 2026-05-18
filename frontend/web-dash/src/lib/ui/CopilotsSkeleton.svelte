<div class="mx-auto max-w-4xl animate-pulse space-y-6 p-8">
	<!-- 2x2 card grid -->
	<div class="grid grid-cols-2 gap-4">
		{#each [1, 2, 3, 4] as _}
			<div class="space-y-4 rounded-[var(--radius-default)] bg-surface-container p-6">
				<div class="h-5 w-3/4 rounded bg-surface-container-high"></div>
				<div class="space-y-2">
					<div class="h-3 w-full rounded bg-surface-container-high"></div>
					<div class="h-3 w-5/6 rounded bg-surface-container-high"></div>
					<div class="h-3 w-4/6 rounded bg-surface-container-high"></div>
				</div>
			</div>
		{/each}
	</div>
</div>
