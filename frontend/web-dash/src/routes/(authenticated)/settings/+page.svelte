<script lang="ts">
	import { page } from '$app/stores';
	import { Button } from '$lib/ui';
	import SettingsSidebar from '$lib/ui/SettingsSidebar.svelte';

	let currentPath = $derived($page.url.pathname);
	let section = $derived(currentPath.replace('/settings', '').replace('/', '') || 'profile');

	function handleSelect(id: string) {
		window.location.href = id === 'profile' ? '/settings' : `/settings/${id}`;
	}

	function handleLogout() {
		window.location.href = '/auth/login';
	}

	// Stub user data for profile form
	let profile = $state({
		name: 'John Doe',
		email: 'john@example.com',
		password: ''
	});

	function handleProfileSave(e: Event) {
		e.preventDefault();
		// TODO: wire to API
		console.log('Profile saved:', profile);
	}
</script>

<div class="flex h-full flex-col bg-surface lg:flex-row">
	<!-- Sidebar -->
	<SettingsSidebar activeId={section} onselect={handleSelect} />

	<!-- Content area -->
	<div class="hidden flex-1 overflow-y-auto lg:block">
		{#if section === 'profile' || section === ''}
			<div class="mx-auto max-w-xl p-6">
				<h2 class="mb-6 font-heading text-2xl font-bold text-on-surface">Profile</h2>

				<div class="mb-6 flex items-center gap-4">
					<div class="flex h-16 w-16 items-center justify-center rounded-full bg-surface-container-high font-heading text-xl font-bold text-on-surface">
						{profile.name
							.split(' ')
							.map((w) => w[0])
							.slice(0, 2)
							.join('')
							.toUpperCase()}
					</div>
					<div>
						<p class="font-heading text-sm font-semibold text-on-surface">Avatar</p>
						<p class="text-xs text-on-surface-variant">Read-only for MVP</p>
					</div>
				</div>

				<form class="space-y-4" onsubmit={handleProfileSave}>
					<div class="space-y-1">
						<label class="text-sm font-medium text-on-surface">Name</label>
						<input
							type="text"
							bind:value={profile.name}
							class="w-full rounded-[var(--radius-default)] border border-outline-variant bg-surface-container px-3 py-2 text-sm text-on-surface outline-none focus:border-primary"
						/>
					</div>
					<div class="space-y-1">
						<label class="text-sm font-medium text-on-surface">Email</label>
						<input
							type="email"
							bind:value={profile.email}
							class="w-full rounded-[var(--radius-default)] border border-outline-variant bg-surface-container px-3 py-2 text-sm text-on-surface outline-none focus:border-primary"
						/>
					</div>
					<div class="space-y-1">
						<label class="text-sm font-medium text-on-surface">Password</label>
						<input
							type="password"
							placeholder="••••••••"
							bind:value={profile.password}
							class="w-full rounded-[var(--radius-default)] border border-outline-variant bg-surface-container px-3 py-2 text-sm text-on-surface outline-none focus:border-primary"
						/>
					</div>
					<div class="flex items-center justify-between pt-2">
						<Button type="submit" variant="primary" size="md">Save changes</Button>
						<Button variant="danger" size="md" onclick={handleLogout}>Log out</Button>
					</div>
				</form>
			</div>
		{:else if section === 'notifications'}
			<div class="mx-auto max-w-xl p-6">
				<h2 class="mb-4 font-heading text-2xl font-bold text-on-surface">Notifications</h2>
				<p class="text-on-surface-variant">Manage your notification preferences.</p>
				<p class="mt-4 text-sm text-on-surface-variant/70">Coming soon</p>
			</div>
		{:else if section === 'team'}
			<div class="mx-auto max-w-xl p-6">
				<h2 class="mb-4 font-heading text-2xl font-bold text-on-surface">Team</h2>
				<p class="text-on-surface-variant">Manage team members and roles.</p>
				<p class="mt-4 text-sm text-on-surface-variant/70">Coming soon</p>
			</div>
		{:else if section === 'channels'}
			<div class="mx-auto max-w-xl p-6">
				<h2 class="mb-4 font-heading text-2xl font-bold text-on-surface">Channels</h2>
				<p class="text-on-surface-variant">Configure WhatsApp, Facebook, and Instagram connections.</p>
				<p class="mt-4 text-sm text-on-surface-variant/70">Coming soon</p>
			</div>
		{:else if section === 'billing'}
			<div class="mx-auto max-w-xl p-6">
				<h2 class="mb-4 font-heading text-2xl font-bold text-on-surface">Billing</h2>
				<p class="text-on-surface-variant">View subscription plan, invoices, and usage.</p>
				<p class="mt-4 text-sm text-on-surface-variant/70">Coming soon</p>
			</div>
		{/if}
	</div>
</div>
