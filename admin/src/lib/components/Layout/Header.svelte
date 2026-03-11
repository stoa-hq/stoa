<script lang="ts">
	import { authStore } from '$lib/stores/auth';
	import { authApi } from '$lib/api/auth';
	import { goto } from '$app/navigation';
	import { base } from '$app/paths';
	import { locale, t } from 'svelte-i18n';

	interface Props {
		title: string;
	}
	let { title }: Props = $props();

	const LOCALE_LABELS: Record<string, string> = {
		'de-DE': 'Deutsch',
		'en-US': 'English'
	};

	let langOpen = $state(false);

	function setLocale(loc: string) {
		locale.set(loc);
		localStorage.setItem('stoa_admin_locale', loc);
		langOpen = false;
	}

	function handleClickOutside(event: MouseEvent) {
		const target = event.target as HTMLElement;
		if (!target.closest('.lang-switcher')) {
			langOpen = false;
		}
	}

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

<svelte:document onclick={handleClickOutside} />

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

		<div class="relative lang-switcher">
			<button
				onclick={() => langOpen = !langOpen}
				class="btn btn-secondary btn-sm flex items-center gap-1.5"
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<circle cx="12" cy="12" r="10" stroke-width="1.5" />
					<path stroke-width="1.5" d="M2 12h20M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10A15.3 15.3 0 0 1 12 2z" />
				</svg>
				{LOCALE_LABELS[$locale ?? 'de-DE'] ?? $locale}
			</button>
			{#if langOpen}
				<div class="absolute right-0 mt-1 w-36 bg-white border border-gray-200 rounded-lg shadow-lg z-50 py-1">
					{#each Object.entries(LOCALE_LABELS) as [loc, label]}
						<button
							onclick={() => setLocale(loc)}
							class="w-full text-left px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-100 flex items-center justify-between"
						>
							{label}
							{#if $locale === loc}
								<span class="text-primary-600 font-semibold">&#10003;</span>
							{/if}
						</button>
					{/each}
				</div>
			{/if}
		</div>

		<button class="btn btn-secondary btn-sm" onclick={handleLogout}>{$t('common.logout')}</button>
	</div>
</header>
