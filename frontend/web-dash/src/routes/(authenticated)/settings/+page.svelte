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
	<h1 class="mb-6 font-heading text-2xl font-bold text-text dark:text-text-dark">Settings</h1>

	<Card>
		<h2 class="mb-4 font-heading text-lg font-semibold text-text dark:text-text-dark">Profile</h2>
		<p class="text-text/60 dark:text-text-dark/60">Profile information will be displayed here.</p>
	</Card>

	<Card class="mt-4">
		<h2 class="mb-4 font-heading text-lg font-semibold text-text dark:text-text-dark">Theme</h2>
		<p class="mb-3 text-sm text-text/60 dark:text-text-dark/60">
			Current: <span class="font-semibold text-text dark:text-text-dark">{currentMode}</span>
			(effective: <span class="font-semibold text-text dark:text-text-dark">{effectiveMode}</span>)
		</p>
		<div class="flex gap-2">
			<Button variant={currentMode === 'system' ? 'primary' : 'ghost'} size="sm" onclick={() => theme.setMode('system')}>System</Button>
			<Button variant={currentMode === 'light' ? 'primary' : 'ghost'} size="sm" onclick={() => theme.setMode('light')}>Light</Button>
			<Button variant={currentMode === 'dark' ? 'primary' : 'ghost'} size="sm" onclick={() => theme.setMode('dark')}>Dark</Button>
		</div>
	</Card>

	<Card class="mt-4">
		<h2 class="mb-4 font-heading text-lg font-semibold text-text dark:text-text-dark">Account</h2>
		<Button variant="danger" size="md" onclick={handleLogout}>Log out</Button>
	</Card>
</div>