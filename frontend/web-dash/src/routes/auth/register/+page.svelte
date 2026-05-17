<script lang="ts">
	import { Button, Card, Input } from '$lib/ui';
	import { deserialize } from '$app/forms';

	let tenantName = $state('');
	let email = $state('');
	let password = $state('');
	let error = $state('');
	let loading = $state(false);

	async function handleSubmit(e: Event) {
		e.preventDefault();
		error = '';
		loading = true;

		const form = e.target as HTMLFormElement;
		const formData = new FormData(form);

		try {
			const response = await fetch('?/register', {
				method: 'POST',
				body: formData
			});

			const text = await response.text();
			const wrapper = JSON.parse(text);
			const result = deserialize(wrapper.data);

			if (result.success) {
				if (result.access_token) {
					window.location.href = `/?access_token=${encodeURIComponent(result.access_token)}`;
				} else {
					window.location.href = '/auth/login';
				}
			} else {
				error = result.error || 'Registration failed';
			}
		} catch {
			error = 'Registration failed. Please try again.';
		} finally {
			loading = false;
		}
	}
</script>

<div class="flex min-h-screen items-center justify-center bg-surface p-4">
	<Card class="w-full max-w-md">
		<h1 class="mb-6 text-center font-heading text-2xl font-bold text-on-surface">Create your workspace</h1>

		{#if error}
			<div class="mb-4 rounded-[var(--radius-default)] bg-error/10 p-3 text-sm text-on-error-container">{error}</div>
		{/if}

		<form method="POST" onsubmit={handleSubmit}>
			<div class="mb-4">
				<label for="tenant_name" class="mb-1 block font-heading text-xs font-semibold tracking-wide text-on-surface">Workspace name</label>
				<Input type="text" bind:value={tenantName} name="tenant_name" placeholder="Acme Corp" required />
			</div>
			<div class="mb-4">
				<label for="email" class="mb-1 block font-heading text-xs font-semibold tracking-wide text-on-surface">Email</label>
				<Input type="email" bind:value={email} name="email" placeholder="admin@acme.com" required />
			</div>
			<div class="mb-6">
				<label for="password" class="mb-1 block font-heading text-xs font-semibold tracking-wide text-on-surface">Password</label>
				<Input type="password" bind:value={password} name="password" placeholder="••••••••" required />
			</div>
			<Button type="submit" variant="primary" size="lg" class="w-full" disabled={loading}>
				{loading ? 'Creating...' : 'Create workspace'}
			</Button>
		</form>

		<p class="mt-4 text-center text-sm text-on-surface/50">
			Already have an account?
			<a href="/auth/login" class="text-secondary hover:underline">Sign in</a>
		</p>
	</Card>
</div>