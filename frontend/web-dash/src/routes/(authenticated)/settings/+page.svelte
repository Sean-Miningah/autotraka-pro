<script lang="ts">
	import { theme } from '$lib/stores/theme';
	import { Button, Card } from '$lib/ui';

	let currentMode = $derived($theme.mode);
	let effectiveMode = $derived($theme.effective);

	function handleLogout() {
		window.location.href = '/auth/login';
	}
</script>

<div class="mx-auto max-w-lg p-8">
	<h1 class="mb-6 font-heading text-2xl font-bold text-on-surface">Settings</h1>

	<Card>
		<h2 class="mb-4 font-heading text-lg font-semibold text-on-surface">Profile</h2>
		<p class="text-on-surface/50">Profile information will be displayed here.</p>
	</Card>

	<Card class="mt-4">
		<h2 class="mb-4 font-heading text-lg font-semibold text-on-surface">Theme</h2>
		<p class="mb-3 text-sm text-on-surface/50">
			Current: <span class="font-semibold text-on-surface">{currentMode}</span>
			(effective: <span class="font-semibold text-on-surface">{effectiveMode}</span>)
		</p>
		<div class="flex gap-2">
			<Button variant={currentMode === 'system' ? 'primary' : 'outline'} size="sm" onclick={() => theme.setMode('system')}>System</Button>
			<Button variant={currentMode === 'light' ? 'primary' : 'outline'} size="sm" onclick={() => theme.setMode('light')}>Light</Button>
		</div>
	</Card>

	<Card class="mt-4">
		<h2 class="mb-4 font-heading text-lg font-semibold text-on-surface">Account</h2>
		<Button variant="danger" size="md" onclick={handleLogout}>Log out</Button>
	</Card>
</div>