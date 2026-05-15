<script lang="ts">
	type Status = 'open' | 'pending' | 'escalated' | 'resolved' | 'closed';
	type HandledBy = 'ai' | 'human' | 'hybrid';

	interface Props {
		status?: Status | null;
		handledBy?: HandledBy | null;
		class?: string;
	}

	let { status = null, handledBy = null, class: className = '' }: Props = $props();

	const statusClasses: Record<Status, string> = {
		open: 'bg-success text-text',
		pending: 'bg-primary text-text',
		escalated: 'bg-secondary text-text',
		resolved: 'bg-surface text-text dark:bg-surface-dark',
		closed: 'bg-surface text-text/60 dark:bg-surface-dark'
	};

	const handledByClasses: Record<HandledBy, string> = {
		ai: 'bg-whatsapp text-text',
		human: 'bg-primary text-text',
		hybrid: 'bg-secondary text-text'
	};
</script>

<div class="inline-flex items-center gap-2 {className}">
	{#if status}
		<span class="inline-flex items-center rounded-sm border-2 border-text px-2 py-0.5 text-xs font-heading font-semibold uppercase {statusClasses[status]}">
			{status}
		</span>
	{/if}
	{#if handledBy}
		<span class="inline-flex items-center rounded-sm border-2 border-text px-2 py-0.5 text-xs font-heading font-semibold uppercase {handledByClasses[handledBy]}">
			{handledBy}
		</span>
	{/if}
</div>