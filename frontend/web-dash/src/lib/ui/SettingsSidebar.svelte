<script lang="ts">
	interface Section {
		id: string;
		label: string;
	}

	interface Props {
		activeId: string;
		onselect: (id: string) => void;
	}

	let { activeId, onselect }: Props = $props();

	const sections: Section[] = [
		{ id: 'profile', label: 'Profile' },
		{ id: 'notifications', label: 'Notifications' },
		{ id: 'team', label: 'Team' },
		{ id: 'channels', label: 'Channels' },
		{ id: 'billing', label: 'Billing' }
	];
</script>

<div class="flex h-full w-[280px] flex-col border-r border-outline-variant bg-surface">
	<div class="border-b border-outline-variant bg-surface-container px-4 py-3">
		<h2 class="font-heading text-xl font-bold text-on-surface">Settings</h2>
	</div>

	<nav class="flex-1 overflow-y-auto">
		<div class="divide-y divide-outline-variant/30">
			{#each sections as section (section.id)}
				<button
					class="flex w-full items-center gap-3 px-4 py-3 text-left transition-colors {activeId === section.id
						? 'border-l-[4px] border-l-primary bg-primary-container'
						: 'hover:bg-surface-container-high border-l-[4px] border-l-transparent'}"
					onclick={() => onselect(section.id)}
				>
					<span class="font-heading text-sm font-semibold text-on-surface">{section.label}</span>
				</button>
			{/each}
		</div>
	</nav>
</div>
