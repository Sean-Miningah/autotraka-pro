<script lang="ts">
	type ChannelType = 'whatsapp' | 'instagram' | 'facebook';

	interface Props {
		channel: ChannelType;
		class?: string;
	}

	let { channel, class: className = '' }: Props = $props();

	const channelConfig: Record<ChannelType, { label: string; bg: string; textColor: string }> = {
		whatsapp: { label: 'WhatsApp', bg: 'bg-whatsapp/10', textColor: 'text-[#005523]' },
		instagram: { label: 'Instagram', bg: 'bg-instagram/10', textColor: 'text-[#a02b47]' },
		facebook: { label: 'Facebook', bg: 'bg-facebook/10', textColor: 'text-[#0e4ea3]' }
	};

	const config = $derived(channelConfig[channel]);
</script>

<span class="inline-flex items-center gap-1 rounded-[var(--radius-full)] px-2.5 py-0.5 text-xs font-heading font-semibold {config.bg} {config.textColor} {className}">
	{config.label}
</span>