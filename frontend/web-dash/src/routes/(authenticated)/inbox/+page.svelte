<script lang="ts">
	import { conversations } from '$lib/stores/conversations';
	import { StatusPill, ChannelBadge, Badge, Button, Drawer } from '$lib/ui';
	import { formatRelativeTime } from '$lib/utils/format';
	import type { EnrichedConversation } from '$lib/stores/conversations';

	let statusFilter = $state<string>('open');
	let handledByFilter = $state<string | null>(null);
	let showFilterDrawer = $state(false);
	let convList = $state<EnrichedConversation[]>([]);
	let loading = $state(false);
	let error = $state<string | null>(null);
	let hasMore = $state(true);

	const statusTabs: { value: string; label: string }[] = [
		{ value: 'open', label: 'Open' },
		{ value: 'pending', label: 'Pending' },
		{ value: 'escalated', label: 'Escalated' },
		{ value: 'resolved', label: 'Resolved' }
	];

	const handledByOptions: { value: string | null; label: string }[] = [
		{ value: null, label: 'All' },
		{ value: 'ai', label: 'AI' },
		{ value: 'human', label: 'Human' },
		{ value: 'hybrid', label: 'Hybrid' }
	];

	async function fetchConversations() {
		conversations.setStatusFilter(statusFilter);
		conversations.setHandledByFilter(handledByFilter);
		await conversations.fetchInitial();
		syncState();
	}

	function syncState() {
		let state: { conversations: EnrichedConversation[]; loading: boolean; error: string | null; hasMore: boolean } | undefined;
		conversations.subscribe((s) => { state = s; })();
		if (state) {
			convList = state.conversations;
			loading = state.loading;
			error = state.error;
			hasMore = state.hasMore;
		}
	}

	function parseLastMessage(raw: string): string {
		if (!raw) return 'No messages yet';
		return conversations.parseLastMessage(raw);
	}

	async function handleLoadMore() {
		await conversations.loadMore();
		syncState();
	}

	function navigateToConversation(id: string) {
		window.location.href = `/inbox/${id}`;
	}

	$effect(() => {
		fetchConversations();
	});
</script>

<div class="flex h-screen flex-col bg-surface">
	<!-- Header with status tabs -->
	<div class="border-b border-outline-variant bg-surface-container">
		<div class="flex items-center justify-between px-4 py-3">
			<h1 class="font-heading text-xl font-bold text-on-surface">Inbox</h1>
			<Button variant="outline" size="sm" onclick={() => showFilterDrawer = true}>Filter</Button>
		</div>
		<div class="flex gap-1 overflow-x-auto px-4 pb-2">
			{#each statusTabs as tab (tab.label)}
				<button
					class="px-3 py-1.5 font-heading text-xs font-semibold border-b-2 transition-colors {statusFilter === tab.value
						? 'border-primary text-on-primary-container'
						: 'border-transparent text-on-surface-variant hover:border-outline-variant'}"
					onclick={() => { statusFilter = tab.value; fetchConversations(); }}
				>
					{tab.label}
				</button>
			{/each}
		</div>
	</div>

	<!-- Conversation list -->
	<div class="flex-1 overflow-y-auto pb-20 lg:pb-4">
		{#if loading && convList.length === 0}
			<div class="flex items-center justify-center p-8">
				<p class="font-heading text-on-surface-variant">Loading conversations...</p>
			</div>
		{:else if error}
			<div class="flex items-center justify-center p-8">
				<p class="font-heading text-error">{error}</p>
			</div>
		{:else if convList.length === 0}
			<div class="flex items-center justify-center p-8">
				<p class="font-heading text-on-surface-variant">No conversations found.</p>
			</div>
		{:else}
			<div class="divide-y divide-outline-variant/30">
				{#each convList as conv (conv.id)}
					<button
						class="flex w-full items-start gap-3 border-b border-outline-variant/30 px-4 py-3 text-left transition-colors hover:bg-surface-container-high border-l-[4px] border-l-primary"
						onclick={() => navigateToConversation(conv.id)}
					>
						<div class="min-w-0 flex-1">
							<div class="flex items-center justify-between gap-2">
								<span class="truncate font-heading text-sm font-semibold text-on-surface">
									{conv.contact_name || 'Unknown'}
								</span>
								<div class="flex shrink-0 items-center gap-1.5">
									{#if conv.channel_type}
										<ChannelBadge channel={conv.channel_type as 'whatsapp' | 'instagram' | 'facebook'} />
									{/if}
									<StatusPill status={(conv.status || 'open') as 'open' | 'pending' | 'escalated' | 'resolved' | 'closed'} handledBy={conv.handled_by as 'ai' | 'human' | 'hybrid'} />
								</div>
							</div>
							<p class="mt-0.5 truncate text-sm text-on-surface-variant">
								{parseLastMessage(conv.last_message)}
							</p>
							<div class="mt-1 flex items-center justify-between">
								<span class="text-xs text-on-surface-variant">
									{formatRelativeTime(conv.updated_at)}
								</span>
								{#if conv.unread_count > 0}
									<Badge variant="tonal">{conv.unread_count}</Badge>
								{/if}
							</div>
						</div>
					</button>
				{/each}
			</div>

			{#if hasMore}
				<div class="flex justify-center py-4">
					<Button variant="outline" size="sm" onclick={handleLoadMore} disabled={loading}>
						{loading ? 'Loading...' : 'Load more'}
					</Button>
				</div>
			{/if}
		{/if}
	</div>
</div>

<!-- Advanced filter drawer -->
<Drawer open={showFilterDrawer} onclose={() => showFilterDrawer = false}>
	<div class="p-4">
		<h2 class="mb-4 font-heading text-lg font-bold text-on-surface">Filters</h2>

		<div class="mb-6">
			<h3 class="mb-2 font-heading text-sm font-semibold text-on-surface">Handled by</h3>
			<div class="flex flex-wrap gap-2">
				{#each handledByOptions as option (option.label)}
					<button
						class="rounded-[var(--radius-default)] px-3 py-1.5 font-heading text-xs font-semibold transition-colors {handledByFilter === option.value
							? 'bg-primary/10 text-on-primary-container'
							: 'text-on-surface-variant hover:bg-surface-container-high'}"
						onclick={() => { handledByFilter = option.value; showFilterDrawer = false; fetchConversations(); }}
					>
						{option.label}
					</button>
				{/each}
			</div>
		</div>

		<Button variant="outline" size="sm" onclick={() => { handledByFilter = null; statusFilter = 'open'; showFilterDrawer = false; fetchConversations(); }}>
			Reset filters
		</Button>
	</div>
</Drawer>