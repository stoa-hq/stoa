<script lang="ts">
	import { authApi } from '$lib/api/auth';
	import { authStore } from '$lib/stores/auth';
	import { notifications } from '$lib/stores/notifications';
	import { goto } from '$app/navigation';
	import { base } from '$app/paths';
	import { ApiClientError } from '$lib/api/client';
	import { t } from 'svelte-i18n';
	import Logo from '$lib/components/Logo.svelte';

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
				notifications.error(err.errors[0]?.detail ?? $t('login.failed'));
			} else {
				notifications.error($t('login.connectionError'));
			}
		} finally {
			loading = false;
		}
	}
</script>

<div class="min-h-screen flex items-center justify-center bg-[var(--bg)]">
	<div class="w-full max-w-sm p-8 bg-[var(--surface)] dark:bg-[#1A1A2E]/90 dark:backdrop-blur-xl rounded-xl shadow-xl border border-[var(--card-border)]">
		<div class="text-center mb-8">
			<div class="flex justify-center mb-3">
				<Logo />
			</div>
			<p class="text-sm text-[var(--text-muted)] mt-1">{$t('login.subtitle')}</p>
		</div>

		<form onsubmit={handleSubmit} class="space-y-4">
			<div>
				<label class="label" for="email">{$t('login.email')}</label>
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
				<label class="label" for="password">{$t('login.password')}</label>
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
				{loading ? $t('login.submitting') : $t('login.submit')}
			</button>
		</form>
	</div>
</div>
