<div class="flex h-full animate-pulse">
	<!-- Left sidebar: customer list -->
	<div class="w-1/4 border-r border-outline-variant">
		<div class="border-b border-outline-variant bg-surface-container p-4">
			<div class="h-5 w-24 rounded bg-surface-container-high"></div>
		</div>
		<div class="divide-y divide-outline-variant/30">
			{#each [1, 2, 3, 4, 5] as _}
				<div class="flex items-center gap-3 p-3">
					<div class="h-10 w-10 rounded-full bg-surface-container-high"></div>
					<div class="flex-1 space-y-2">
						<div class="h-3 w-3/4 rounded bg-surface-container-high"></div>
						<div class="h-3 w-1/2 rounded bg-surface-container-high"></div>
					</div>
				</div>
			{/each}
		</div>
	</div>

	<!-- Right detail area -->
	<div class="flex-1 p-6">
		<div class="rounded-[var(--radius-default)] bg-surface-container p-6">
			<div class="mb-6 flex items-center gap-4">
				<div class="h-16 w-16 rounded-full bg-surface-container-high"></div>
				<div class="space-y-2">
					<div class="h-5 w-32 rounded bg-surface-container-high"></div>
					<div class="h-3 w-48 rounded bg-surface-container-high"></div>
				</div>
			</div>
			<div class="space-y-4">
				{#each [1, 2, 3] as _}
					<div class="space-y-1">
						<div class="h-3 w-20 rounded bg-surface-container-high"></div>
						<div class="h-8 w-full rounded bg-surface-container-high"></div>
					</div>
				{/each}
			</div>
		</div>
	</div>
</div>
