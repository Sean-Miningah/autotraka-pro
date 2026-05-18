<script lang="ts">
	import { ChannelBadge } from '$lib/ui';

	interface Customer {
		id: string;
		name: string;
		phone?: string;
		channel_type: string;
		lastMessage: string;
	}

	interface Props {
		customers: Customer[];
		activeId: string | null;
		onselect: (id: string) => void;
	}

	let { customers, activeId, onselect }: Props = $props();
	let searchQuery = $state('');

	let filtered = $derived(
		searchQuery.trim() === ''
			? customers
			: customers.filter((c) =>
					c.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
					c.lastMessage.toLowerCase().includes(searchQuery.toLowerCase())
			)
	);

	function initials(name: string): string {
		return name
			.split(' ')
			.map((w) => w[0])
			.slice(0, 2)
			.join('')
			.toUpperCase();
	}
</script>

<div class="flex h-full w-[280px] flex-col border-r border-outline-variant bg-surface">
	<!-- Search -->
	<div class="border-b border-outline-variant p-3">
		<input
			type="text"
			placeholder="Search customers..."
			bind:value={searchQuery}
			class="w-full rounded-[var(--radius-default)] bg-surface-container px-3 py-2 text-sm text-on-surface placeholder:text-on-surface-variant/50 outline-none focus:ring-2 focus:ring-primary/30"
		/>
	</div>

	<!-- Customer list -->
	<div class="flex-1 overflow-y-auto">
		{#if filtered.length === 0}
			<div class="p-4 text-center text-sm text-on-surface-variant">
				No customers found.
			</div>
		{:else}
			<div class="divide-y divide-outline-variant/30">
				{#each filtered as customer (customer.id)}
					<button
						class="flex w-full items-start gap-3 px-4 py-3 text-left transition-colors {activeId === customer.id
							? 'border-l-[4px] border-l-primary bg-primary-container'
							: 'hover:bg-surface-container-high border-l-[4px] border-l-transparent'}"
						onclick={() => onselect(customer.id)}
					>
						<div
							class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-surface-container-high font-heading text-xs font-bold text-on-surface"
						>
							{initials(customer.name)}
						</div>
						<div class="min-w-0 flex-1">
							<div class="flex items-center justify-between gap-2">
								<span class="truncate font-heading text-sm font-semibold text-on-surface">
									{customer.name}
								</span>
								{#if customer.channel_type}
									<ChannelBadge channel={customer.channel_type as 'whatsapp' | 'instagram' | 'facebook'} />
								{/if}
							</div>
							<p class="mt-0.5 truncate text-sm text-on-surface-variant">
								{customer.lastMessage}
							</p>
						</div>
					</button>
				{/each}
			</div>
		{/if}
	</div>
</div>
