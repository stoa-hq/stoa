<script lang="ts">
	import { authStore } from '$lib/stores/auth';
	import { theme } from '$lib/stores/theme';
	import { authApi } from '$lib/api/auth';
	import { goto } from '$app/navigation';
	import { base } from '$app/paths';
	import { locale, t } from 'svelte-i18n';
	import { Sun, Moon, Globe, LogOut, Menu } from 'lucide-svelte';

	interface Props {
		title: string;
		onMenuClick?: () => void;
	}
	let { title, onMenuClick }: Props = $props();

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

<header class="h-14 backdrop-blur-xl bg-[var(--header-bg)] border-b border-[var(--card-border)] flex items-center justify-between px-4 md:px-6 shrink-0 sticky top-0 z-30">
	<div class="flex items-center gap-3">
		{#if onMenuClick}
			<button class="md:hidden p-1.5 rounded-lg text-[var(--text-muted)] hover:text-[var(--text)] hover:bg-gray-100 dark:hover:bg-white/5 transition-colors" onclick={onMenuClick}>
				<Menu class="w-5 h-5" />
			</button>
		{/if}
		<h1 class="text-base font-semibold text-[var(--text)]">{title}</h1>
	</div>

	<div class="flex items-center gap-2">
		<!-- Theme Toggle -->
		<button
			class="p-2 rounded-lg text-[var(--text-muted)] hover:text-[var(--text)] hover:bg-gray-100 dark:hover:bg-white/5 transition-colors"
			onclick={() => theme.toggle()}
			title={$t('common.theme')}
		>
			{#if $theme === 'dark'}
				<Sun class="w-4 h-4" />
			{:else}
				<Moon class="w-4 h-4" />
			{/if}
		</button>

		<!-- Language Switcher -->
		<div class="relative lang-switcher">
			<button
				onclick={() => langOpen = !langOpen}
				class="p-2 rounded-lg text-[var(--text-muted)] hover:text-[var(--text)] hover:bg-gray-100 dark:hover:bg-white/5 transition-colors"
			>
				<Globe class="w-4 h-4" />
			</button>
			{#if langOpen}
				<div class="absolute right-0 mt-1 w-36 bg-[var(--card)] border border-[var(--card-border)] rounded-lg shadow-lg backdrop-blur-xl z-50 py-1">
					{#each Object.entries(LOCALE_LABELS) as [loc, label]}
						<button
							onclick={() => setLocale(loc)}
							class="w-full text-left px-3 py-1.5 text-sm text-[var(--text)] hover:bg-gray-100 dark:hover:bg-white/5 flex items-center justify-between transition-colors"
						>
							{label}
							{#if $locale === loc}
								<span class="text-primary-500 font-semibold">&#10003;</span>
							{/if}
						</button>
					{/each}
				</div>
			{/if}
		</div>

		<!-- User -->
		{#if $authStore.user}
			<div class="hidden sm:flex items-center gap-2 text-sm">
				<div class="w-7 h-7 rounded-full bg-primary-600 flex items-center justify-center text-white text-xs font-bold">
					{$authStore.user.email[0].toUpperCase()}
				</div>
				<span class="text-[var(--text-muted)] text-xs">{$authStore.user.email}</span>
			</div>
		{/if}

		<!-- Logout -->
		<button
			class="p-2 rounded-lg text-[var(--text-muted)] hover:text-red-500 hover:bg-gray-100 dark:hover:bg-white/5 transition-colors"
			onclick={handleLogout}
			title={$t('common.logout')}
		>
			<LogOut class="w-4 h-4" />
		</button>
	</div>
</header>
