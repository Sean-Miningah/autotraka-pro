<script lang="ts">
	interface CallInfo {
		contactName: string;
		phoneNumber?: string;
		channelId: 'whatsapp' | 'facebook' | 'instagram';
		startedAt: Date;
		isMuted: boolean;
		isOnHold: boolean;
	}

	interface Props {
		call: CallInfo | null;
		onMute?: () => void;
		onHold?: () => void;
		onTransfer?: () => void;
		onEndCall?: () => void;
		onExpand?: () => void;
	}

	let { call, onMute, onHold, onTransfer, onEndCall, onExpand }: Props = $props();

	let expanded = $state(false);
	let elapsed = $state('00:00');
	let intervalId: ReturnType<typeof setInterval> | null = null;

	$effect(() => {
		if (call) {
			intervalId = setInterval(() => {
				const now = new Date();
				const diff = Math.floor((now.getTime() - call.startedAt.getTime()) / 1000);
				const mins = Math.floor(diff / 60).toString().padStart(2, '0');
				const secs = (diff % 60).toString().padStart(2, '0');
				elapsed = `${mins}:${secs}`;
			}, 1000);
		} else {
			if (intervalId) clearInterval(intervalId);
			elapsed = '00:00';
		}
		return () => {
			if (intervalId) clearInterval(intervalId);
		};
	});

	const channelColors: Record<string, string> = {
		whatsapp: 'bg-whatsapp',
		facebook: 'bg-facebook',
		instagram: 'bg-instagram'
	};

	function toggleExpand() {
		expanded = !expanded;
		onExpand?.();
	}
</script>

{#if call}
	<div class="fixed bottom-4 left-1/2 z-50 -translate-x-1/2 rounded-[var(--radius-default)] border border-outline-variant bg-surface-container-lowest shadow-[var(--shadow-elevation-2)] transition-all {expanded ? 'w-80' : 'w-auto'}">
		<button class="flex items-center gap-3 px-4 py-2.5 w-full" onclick={toggleExpand}>
			<span class="flex h-8 w-8 items-center justify-center rounded-[var(--radius-full)] {channelColors[call.channelId] || 'bg-primary'} text-white">
				<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z"/></svg>
			</span>
			<div class="flex flex-col items-start">
				<span class="font-heading text-sm font-semibold text-on-surface">{call.contactName}</span>
				<span class="text-xs text-on-surface-variant">{elapsed}</span>
			</div>
		</button>

		{#if expanded}
			<div class="border-t border-outline-variant px-4 py-3">
				{#if call.phoneNumber}
					<p class="text-xs text-on-surface-variant mb-3">{call.phoneNumber}</p>
				{/if}

				<div class="flex items-center justify-center gap-3">
					<button
						class="flex h-10 w-10 items-center justify-center rounded-[var(--radius-full)] transition-colors {call.isMuted ? 'bg-error text-on-error' : 'bg-surface-container-high text-on-surface hover:bg-surface-container-highest'}"
						onclick={onMute}
						aria-label={call.isMuted ? 'Unmute' : 'Mute'}
					>
						<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 5a6 6 0 0 1 3.9 10.6"/><path d="M6.7 6.7a8 8 0 0 0 3 14.3 8.2 8.2 0 0 0 4.7-1.4"/><path d="M15.5 18.5a8 8 0 0 0 1.8-2.7"/><path d="M2 2l20 20"/></svg>
					</button>

					<button
						class="flex h-10 w-10 items-center justify-center rounded-[var(--radius-full)] transition-colors {call.isOnHold ? 'bg-secondary-container text-on-secondary-container' : 'bg-surface-container-high text-on-surface hover:bg-surface-container-highest'}"
						onclick={onHold}
						aria-label={call.isOnHold ? 'Resume' : 'Hold'}
					>
						<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M9 12h6"/></svg>
					</button>

					<button
						class="flex h-10 w-10 items-center justify-center rounded-[var(--radius-full)] bg-surface-container-high text-on-surface hover:bg-surface-container-highest transition-colors"
						onclick={onTransfer}
						aria-label="Transfer"
					>
						<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14"/><path d="m12 5 7 7-7 7"/></svg>
					</button>

					<button
						class="flex h-10 w-10 items-center justify-center rounded-[var(--radius-full)] bg-error text-on-error hover:opacity-90 transition-opacity"
						onclick={onEndCall}
						aria-label="End call"
					>
						<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z"/></svg>
					</button>
				</div>
			</div>
		{/if}
	</div>
{/if}