<script lang="ts">
	import { authStore } from '$lib/stores/auth';
	import { authApi } from '$lib/api/auth';
	import { goto } from '$app/navigation';
	import { base } from '$app/paths';

	interface Props {
		title: string;
	}
	let { title }: Props = $props();

	async function handleLogout() {
		try {
			await authApi.logout();
		} catch {
			// ignore errors
		}
		authStore.logout();
		goto(`${base}/login`);
	}
</script>

<header class="h-14 bg-white border-b border-gray-200 flex items-center justify-between px-6 shrink-0">
	<h1 class="text-base font-semibold text-gray-800">{title}</h1>

	<div class="flex items-center gap-4">
		{#if $authStore.user}
			<div class="flex items-center gap-2 text-sm">
				<div class="w-7 h-7 rounded-full bg-primary-600 flex items-center justify-center text-white text-xs font-bold">
					{$authStore.user.email[0].toUpperCase()}
				</div>
				<span class="text-gray-700">{$authStore.user.email}</span>
				<span class="badge badge-blue">{$authStore.user.role}</span>
			</div>
		{/if}
		<button class="btn btn-secondary btn-sm" onclick={handleLogout}>Abmelden</button>
	</div>
</header>
