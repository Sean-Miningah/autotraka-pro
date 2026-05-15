<script lang="ts">
	import { Button, Card, Input } from '$lib/ui';
	import { getGatewayUrl } from '$lib/api/config';

	let email = $state('');
	let password = $state('');
	let error = $state('');
	let loading = $state(false);
	let showTenantPicker = $state(false);
	let tenants: { tenant_id: string; tenant_name: string }[] = $state([]);

	async function handleEmailSubmit(e: Event) {
		e.preventDefault();
		error = '';
		loading = true;

		try {
			const response = await fetch(`${getGatewayUrl()}/api/v1/auth/tenants?email=${encodeURIComponent(email)}`);
			if (!response.ok && response.status !== 404) {
				error = 'Failed to look up tenants';
				return;
			}

			if (response.status === 404) {
				error = 'No account found for this email';
				return;
			}

			const body = await response.json();
			tenants = body.data as { tenant_id: string; tenant_name: string }[];

			if (tenants.length === 1) {
				submitLogin(tenants[0].tenant_id);
			} else {
				showTenantPicker = true;
			}
		} catch {
			error = 'Connection error. Please try again.';
		} finally {
			loading = false;
		}
	}

	async function submitLogin(tenantId: string) {
		error = '';
		loading = true;

		const form = document.querySelector('form') as HTMLFormElement;
		const formData = new FormData(form);
		formData.set('tenant_id', tenantId);

		const response = await fetch('?/login', {
			method: 'POST',
			body: formData
		});
		const result = await response.json();

		if (result.type === 'success' && result.data?.success) {
			const accessToken = result.data.access_token;
			window.location.href = `/?access_token=${encodeURIComponent(accessToken)}`;
		} else {
			error = result.data?.error || 'Login failed';
		}
		loading = false;
	}
</script>

<div class="flex min-h-screen items-center justify-center bg-base dark:bg-base-dark p-4">
	<Card class="w-full max-w-md">
		<h1 class="mb-6 text-center font-heading text-2xl font-bold text-text dark:text-text-dark">Sign in to Autotraka</h1>

		{#if error}
			<div class="mb-4 border-2 border-danger bg-danger/10 p-3 text-sm text-danger">{error}</div>
		{/if}

		{#if !showTenantPicker}
			<form method="POST" onsubmit={handleEmailSubmit}>
				<div class="mb-4">
					<label for="email" class="mb-1 block font-heading text-sm font-semibold text-text dark:text-text-dark">Email</label>
					<Input type="email" bind:value={email} name="email" placeholder="you@company.com" required />
				</div>
				<div class="mb-6">
					<label for="password" class="mb-1 block font-heading text-sm font-semibold text-text dark:text-text-dark">Password</label>
					<Input type="password" bind:value={password} name="password" placeholder="••••••••" required />
				</div>
				<Button type="submit" variant="primary" size="lg" class="w-full" disabled={loading}>
					{loading ? 'Signing in...' : 'Continue'}
				</Button>
			</form>
		{:else}
			<h2 class="mb-4 font-heading text-lg font-semibold text-text dark:text-text-dark">Select workspace</h2>
			<div class="space-y-3">
				{#each tenants as tenant (tenant.tenant_id)}
					<button
						class="w-full border-2 border-text bg-surface p-4 text-left shadow-[4px_4px_0px] shadow-text hover:translate-x-[2px] hover:translate-y-[2px] hover:shadow-[2px_2px_0px] dark:border-text-dark dark:bg-surface-dark dark:shadow-text-dark transition-all"
						onclick={() => submitLogin(tenant.tenant_id)}
						disabled={loading}
					>
						<div class="font-heading font-semibold text-text dark:text-text-dark">{tenant.tenant_name}</div>
					</button>
				{/each}
			</div>
			<button class="mt-4 text-sm text-secondary hover:underline" onclick={() => { showTenantPicker = false; error = ''; }}>
				← Use a different email
			</button>
		{/if}

		<p class="mt-4 text-center text-sm text-text/60 dark:text-text-dark/60">
			Don't have an account?
			<a href="/auth/register" class="text-secondary hover:underline">Register</a>
		</p>
	</Card>
</div>