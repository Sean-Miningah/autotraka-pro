<div class="flex h-full animate-pulse">
	<!-- Left pane: conversation list -->
	<div class="w-1/3 border-r border-outline-variant">
		<div class="border-b border-outline-variant bg-surface-container p-4">
			<div class="mb-4 h-5 w-20 rounded bg-surface-container-high"></div>
			<div class="flex gap-2">
				{#each [1, 2, 3, 4] as _}
					<div class="h-7 w-14 rounded bg-surface-container-high"></div>
				{/each}
			</div>
		</div>
		<div class="divide-y divide-outline-variant/30">
			{#each [1, 2, 3, 4, 5, 6, 7, 8] as _}
				<div class="flex items-start gap-3 p-4">
					<div class="h-10 w-10 shrink-0 rounded-full bg-surface-container-high"></div>
					<div class="flex-1 space-y-2">
						<div class="h-4 w-3/4 rounded bg-surface-container-high"></div>
						<div class="h-3 w-1/2 rounded bg-surface-container-high"></div>
					</div>
				</div>
			{/each}
		</div>
	</div>

	<!-- Right pane: empty placeholder -->
	<div class="flex flex-1 items-center justify-center">
		<div class="h-32 w-48 rounded-[var(--radius-default)] bg-surface-container"></div>
	</div>
</div>
