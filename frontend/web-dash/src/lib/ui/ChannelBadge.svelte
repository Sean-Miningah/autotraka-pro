<script lang="ts">
	type ChannelType = 'whatsapp' | 'instagram' | 'facebook';

	interface Props {
		channel: ChannelType;
		class?: string;
	}

	let { channel, class: className = '' }: Props = $props();

	const channelConfig: Record<ChannelType, { label: string; bg: string; textColor: string }> = {
		whatsapp: { label: 'WhatsApp', bg: 'bg-whatsapp', textColor: 'text-text' },
		instagram: { label: 'Instagram', bg: 'bg-instagram', textColor: 'text-white' },
		facebook: { label: 'Facebook', bg: 'bg-facebook', textColor: 'text-white' }
	};

	const config = $derived(channelConfig[channel]);
</script>

<span class="inline-flex items-center gap-1 rounded-sm border-2 border-text px-2 py-0.5 text-xs font-heading font-semibold {config.bg} {config.textColor} {className}">
	{config.label}
</span>