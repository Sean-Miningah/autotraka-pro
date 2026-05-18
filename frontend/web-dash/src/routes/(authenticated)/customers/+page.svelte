<script lang="ts">
	import { page } from '$app/stores';
	import CustomersSidebar from '$lib/ui/CustomersSidebar.svelte';
	import CustomersEmptyState from '$lib/ui/CustomersEmptyState.svelte';
	import { ChannelBadge } from '$lib/ui';

	interface Customer {
		id: string;
		name: string;
		phone?: string;
		channel_type: string;
		lastMessage: string;
		tags?: string[];
	}

	// TODO: replace with real customer store fetch
	let customers: Customer[] = [
		{
			id: 'cust-1',
			name: 'Alice Johnson',
			phone: '+1 555 0101',
			channel_type: 'whatsapp',
			lastMessage: 'Can you send me the invoice?',
			tags: ['VIP', 'Enterprise']
		},
		{
			id: 'cust-2',
			name: 'Bob Smith',
			phone: '+1 555 0102',
			channel_type: 'instagram',
			lastMessage: 'Thanks for the quick reply!',
			tags: ['New']
		},
		{
			id: 'cust-3',
			name: 'Carol White',
			phone: '+1 555 0103',
			channel_type: 'facebook',
			lastMessage: 'When will my order arrive?',
			tags: ['Support']
		},
		{
			id: 'cust-4',
			name: 'David Lee',
			phone: '+1 555 0104',
			channel_type: 'whatsapp',
			lastMessage: 'I need to reschedule the meeting.',
			tags: []
		},
		{
			id: 'cust-5',
			name: 'Emma Brown',
			phone: '+1 555 0105',
			channel_type: 'instagram',
			lastMessage: 'Great service, thank you!',
			tags: ['VIP']
		}
	];

	let activeId = $derived($page.params.id ?? null);
	let selected = $derived(customers.find((c) => c.id === activeId) ?? null);

	function handleSelect(id: string) {
		window.location.href = `/customers/${id}`;
	}

	function initials(name: string): string {
		return name
			.split(' ')
			.map((w) => w[0])
			.slice(0, 2)
			.join('')
			.toUpperCase();
	}
</script>

<div class="flex h-full flex-col bg-surface lg:flex-row">
	<!-- Sidebar -->
	<CustomersSidebar
		{customers}
		activeId={activeId}
		onselect={handleSelect}
	/>

	<!-- Detail area -->
	<div class="hidden flex-1 lg:flex">
		{#if selected}
			<div class="flex h-full flex-col">
				<!-- Profile card -->
				<div class="border-b border-outline-variant bg-surface-container p-6">
					<div class="flex items-center gap-4">
						<div
							class="flex h-14 w-14 items-center justify-center rounded-full bg-surface-container-high font-heading text-lg font-bold text-on-surface"
						>
							{initials(selected.name)}
						</div>
						<div class="flex-1">
							<h2 class="font-heading text-xl font-bold text-on-surface">{selected.name}</h2>
							<div class="mt-1 flex items-center gap-2">
								<span class="text-sm text-on-surface-variant">{selected.phone ?? 'No phone'}</span>
								<ChannelBadge channel={selected.channel_type as 'whatsapp' | 'instagram' | 'facebook'} />
							</div>
							{#if selected.tags && selected.tags.length > 0}
								<div class="mt-2 flex flex-wrap gap-1.5">
									{#each selected.tags as tag}
										<span class="rounded-full bg-surface-container-high px-2.5 py-0.5 text-xs font-medium text-on-surface-variant">
											{tag}
										</span>
									{/each}
								</div>
							{/if}
						</div>
					</div>
				</div>

				<!-- Conversation timeline placeholder -->
				<div class="flex-1 overflow-y-auto p-6">
					<h3 class="mb-4 font-heading text-sm font-semibold text-on-surface-variant">Conversation Timeline</h3>
					<div class="space-y-4">
						{#each [1, 2, 3] as i}
							<div class="flex gap-3">
								<div class="h-8 w-8 shrink-0 rounded-full bg-surface-container-high"></div>
								<div class="flex-1 rounded-[var(--radius-default)] bg-surface-container p-3">
									<div class="h-3 w-3/4 rounded bg-surface-container-high"></div>
									<div class="mt-2 h-3 w-1/2 rounded bg-surface-container-high"></div>
								</div>
							</div>
						{/each}
					</div>
				</div>
			</div>
		{:else}
			<CustomersEmptyState />
		{/if}
	</div>
</div>
