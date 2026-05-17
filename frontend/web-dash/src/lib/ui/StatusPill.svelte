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
		open: 'bg-primary/10 text-on-primary-container',
		pending: 'bg-secondary/10 text-on-secondary-container',
		escalated: 'bg-error/10 text-on-error-container',
		resolved: 'bg-on-surface/5 text-on-surface/60',
		closed: 'bg-on-surface/5 text-on-surface/40'
	};

	const handledByClasses: Record<HandledBy, string> = {
		ai: 'bg-whatsapp/10 text-[#005523]',
		human: 'bg-primary/10 text-on-primary-container',
		hybrid: 'bg-secondary/10 text-on-secondary-container'
	};
</script>

<div class="inline-flex items-center gap-2 {className}">
	{#if status}
		<span class="inline-flex items-center rounded-[var(--radius-sm)] px-2 py-0.5 text-xs font-heading font-semibold uppercase {statusClasses[status]}">
			{status}
		</span>
	{/if}
	{#if handledBy}
		<span class="inline-flex items-center rounded-[var(--radius-sm)] px-2 py-0.5 text-xs font-heading font-semibold uppercase {handledByClasses[handledBy]}">
			{handledBy}
		</span>
	{/if}
</div>