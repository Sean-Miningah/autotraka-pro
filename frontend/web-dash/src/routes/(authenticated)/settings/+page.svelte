<script lang="ts">
	import { Button, Card } from '$lib/ui';

	function handleLogout() {
		window.location.href = '/auth/login';
	}
</script>

<div class="mx-auto max-w-lg bg-surface p-8">
	<h1 class="mb-6 font-heading text-[28px] font-bold leading-tight text-on-surface">Settings</h1>

	<Card>
		<h2 class="mb-4 font-heading text-lg font-semibold text-on-surface">Profile</h2>
		<p class="text-on-surface-variant">Profile information will be displayed here.</p>
	</Card>

	<Card class="mt-4">
		<h2 class="mb-4 font-heading text-lg font-semibold text-on-surface">Theme</h2>
		<p class="text-on-surface-variant">Light mode is the default appearance.</p>
	</Card>

	<Card class="mt-4">
		<h2 class="mb-4 font-heading text-lg font-semibold text-on-surface">Account</h2>
		<Button variant="danger" size="md" onclick={handleLogout}>Log out</Button>
	</Card>
</div>