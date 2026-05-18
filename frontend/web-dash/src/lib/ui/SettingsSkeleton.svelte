<div class="flex h-full animate-pulse">
	<!-- Left sidebar: section list -->
	<div class="w-56 border-r border-outline-variant">
		<div class="border-b border-outline-variant bg-surface-container p-4">
			<div class="h-5 w-16 rounded bg-surface-container-high"></div>
		</div>
		<div class="divide-y divide-outline-variant/30">
			{#each [1, 2, 3, 4, 5] as _}
				<div class="flex items-center gap-3 p-3">
					<div class="h-4 w-4 rounded bg-surface-container-high"></div>
					<div class="h-3 w-24 rounded bg-surface-container-high"></div>
				</div>
			{/each}
		</div>
	</div>

	<!-- Right content area: form-like layout -->
	<div class="flex-1 space-y-6 p-6">
		<div class="h-6 w-32 rounded bg-surface-container-high"></div>
		{#each [1, 2, 3, 4] as _}
			<div class="space-y-2">
				<div class="h-3 w-20 rounded bg-surface-container-high"></div>
				<div class="h-10 w-full rounded bg-surface-container-high"></div>
			</div>
		{/each}
	</div>
</div>
