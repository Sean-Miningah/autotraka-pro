<script lang="ts">
	import { Button, Card, Input } from '$lib/ui';
	import { enhance } from '$app/forms';
	import type { ActionData } from './$types';

	let { form }: { form: ActionData } = $props();

	let email = $state('');
	let password = $state('');
	let loading = $state(false);
	let error = $state('');
	let showTenantPicker = $state(false);
	let tenants: { tenant_id: string; tenant_name: string }[] = $state([]);
	let selectedTenantId = $state('');
	let formElement: HTMLFormElement;
</script>

<div class="flex min-h-screen items-center justify-center bg-surface p-4">
	<Card class="w-full max-w-md">
		<h1 class="mb-6 text-center font-heading text-2xl font-bold text-on-surface">Sign in to Autotraka</h1>

		{#if error || form?.error}
			<div class="mb-4 rounded-[var(--radius-default)] bg-error/10 p-3 text-sm text-on-error-container">{error || form?.error}</div>
		{/if}

		<form
			bind:this={formElement}
			method="POST"
			action="?/login"
			use:enhance={() => {
				loading = true;
				error = '';
				return async ({ result }) => {
					loading = false;

					if (result.type !== 'success') {
						error = 'Something went wrong';
						return;
					}

					const data = result.data;

					if (data?.success) {
						window.location.href = `/?access_token=${encodeURIComponent(data.access_token)}`;
						return;
					}

					if (data?.needsTenantSelection) {
						tenants = data.tenants as typeof tenants;
						showTenantPicker = true;
						return;
					}

					showTenantPicker = false;
					error = data?.error || 'Login failed';
				};
			}}
		>
			<div class="mb-4" class:hidden={showTenantPicker}>
				<label for="email" class="mb-1 block font-heading text-xs font-semibold tracking-wide text-on-surface">Email</label>
				<Input type="email" bind:value={email} name="email" placeholder="you@company.com" required />
			</div>
			<div class="mb-6" class:hidden={showTenantPicker}>
				<label for="password" class="mb-1 block font-heading text-xs font-semibold tracking-wide text-on-surface">Password</label>
				<Input type="password" bind:value={password} name="password" placeholder="••••••••" required />
			</div>

			{#if showTenantPicker}
				<h2 class="mb-4 font-heading text-lg font-semibold text-on-surface">Select workspace</h2>
				<div class="space-y-3">
					{#each tenants as tenant (tenant.tenant_id)}
						<button
							type="button"
							class="w-full bg-surface-container-lowest border border-outline-variant rounded-[var(--radius-default)] shadow-[var(--shadow-elevation-1)] hover:shadow-[var(--shadow-elevation-2)] transition-shadow p-4 text-left"
							onclick={() => { selectedTenantId = tenant.tenant_id; formElement.requestSubmit(); }}
							disabled={loading}
						>
							<div class="font-heading font-semibold text-on-surface">{tenant.tenant_name}</div>
						</button>
					{/each}
				</div>
				<button class="mt-4 text-sm text-secondary hover:underline" onclick={() => { showTenantPicker = false; selectedTenantId = ''; }} type="button">
					← Use a different email
				</button>
			{/if}

			<input type="hidden" name="tenant_id" value={selectedTenantId} />

			{#if !showTenantPicker}
				<Button type="submit" variant="primary" size="lg" class="w-full" disabled={loading}>
					{loading ? 'Signing in...' : 'Continue'}
				</Button>
			{/if}
		</form>

		<p class="mt-4 text-center text-sm text-on-surface/50">
			Don't have an account?
			<a href="/auth/register" class="text-secondary hover:underline">Register</a>
		</p>
	</Card>
</div>