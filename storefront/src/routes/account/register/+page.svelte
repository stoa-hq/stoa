<script lang="ts">
	import { goto } from '$app/navigation';
	import { customersApi } from '$lib/api/customers';
	import { authApi } from '$lib/api/auth';
	import { authStore } from '$lib/stores/auth';

	let form = $state({ email: '', password: '', first_name: '', last_name: '' });
	let error = $state('');
	let loading = $state(false);

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		loading = true;
		error = '';
		try {
			await customersApi.register(form);
			// Auto-login after register
			const loginRes = await authApi.login(form.email, form.password);
			if (loginRes.data) {
				authStore.login(loginRes.data.access_token, loginRes.data.refresh_token);
			}
			goto('/account/orders');
		} catch (err: unknown) {
			error = (err as Error).message ?? 'Registrierung fehlgeschlagen.';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Registrieren – stoa</title>
</svelte:head>

<div class="min-h-[60vh] flex items-center justify-center px-4 py-12">
	<div class="w-full max-w-sm">
		<div class="text-center mb-8">
			<h1 class="text-2xl font-bold text-gray-900">Konto erstellen</h1>
			<p class="text-gray-500 mt-1 text-sm">Bereits registriert?
				<a href="/account/login" class="text-primary-700 font-medium hover:underline">Anmelden</a>
			</p>
		</div>

		<form onsubmit={submit} class="card p-6 space-y-4">
			<div class="grid grid-cols-2 gap-4">
				<div>
					<label class="label" for="first_name">Vorname</label>
					<input class="input" id="first_name" bind:value={form.first_name} required />
				</div>
				<div>
					<label class="label" for="last_name">Nachname</label>
					<input class="input" id="last_name" bind:value={form.last_name} required />
				</div>
			</div>
			<div>
				<label class="label" for="email">E-Mail</label>
				<input class="input" id="email" type="email" bind:value={form.email} required autocomplete="email" />
			</div>
			<div>
				<label class="label" for="password">Passwort</label>
				<input class="input" id="password" type="password" bind:value={form.password} required minlength="8" autocomplete="new-password" />
			</div>

			{#if error}
				<p class="text-red-600 text-sm">{error}</p>
			{/if}

			<button type="submit" disabled={loading} class="btn btn-primary w-full">
				{#if loading}
					<svg class="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
						<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"/>
					</svg>
				{:else}
					Registrieren
				{/if}
			</button>
		</form>
	</div>
</div>
