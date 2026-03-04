<script lang="ts">
	import { authApi } from '$lib/api/auth';
	import { authStore } from '$lib/stores/auth';
	import { notifications } from '$lib/stores/notifications';
	import { goto } from '$app/navigation';
	import { base } from '$app/paths';
	import { ApiClientError } from '$lib/api/client';

	let email = $state('');
	let password = $state('');
	let loading = $state(false);

	async function handleSubmit(e: Event) {
		e.preventDefault();
		loading = true;
		try {
			const res = await authApi.login(email, password);
			authStore.setTokens(res.data.access_token, res.data.refresh_token);
			goto(`${base}/`);
		} catch (err) {
			if (err instanceof ApiClientError) {
				notifications.error(err.errors[0]?.detail ?? 'Login fehlgeschlagen');
			} else {
				notifications.error('Verbindungsfehler');
			}
		} finally {
			loading = false;
		}
	}
</script>

<div class="min-h-screen flex items-center justify-center bg-gray-100">
	<div class="card w-full max-w-sm p-8">
		<div class="text-center mb-8">
			<h1 class="text-2xl font-bold text-gray-900">Commerce Admin</h1>
			<p class="text-sm text-gray-500 mt-1">Melden Sie sich an</p>
		</div>

		<form onsubmit={handleSubmit} class="space-y-4">
			<div>
				<label class="label" for="email">E-Mail</label>
				<input
					id="email"
					type="email"
					class="input"
					bind:value={email}
					required
					autocomplete="email"
					placeholder="admin@example.com"
				/>
			</div>
			<div>
				<label class="label" for="password">Passwort</label>
				<input
					id="password"
					type="password"
					class="input"
					bind:value={password}
					required
					autocomplete="current-password"
				/>
			</div>
			<button type="submit" class="btn btn-primary w-full" disabled={loading}>
				{loading ? 'Anmelden…' : 'Anmelden'}
			</button>
		</form>
	</div>
</div>
