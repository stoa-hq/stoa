<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth';
	import { apiKeysApi, type StoreAPIKey, type CreateKeyResponse } from '$lib/api/api-keys';
	import { t } from 'svelte-i18n';
	import { fmt } from '$lib/i18n/formatters';

	const MAX_KEYS = 5;

	const ALL_PERMISSIONS = [
		{ value: 'store.products.read', labelKey: 'apiKeys.permProductRead' },
		{ value: 'store.cart.manage', labelKey: 'apiKeys.permCartManage' },
		{ value: 'store.checkout', labelKey: 'apiKeys.permCheckout' },
		{ value: 'store.account.read', labelKey: 'apiKeys.permAccountRead' },
		{ value: 'store.account.update', labelKey: 'apiKeys.permAccountUpdate' },
		{ value: 'store.orders.read', labelKey: 'apiKeys.permOrdersRead' }
	];

	let keys = $state<StoreAPIKey[]>([]);
	let loading = $state(true);
	let error = $state('');

	// Modal state
	let showModal = $state(false);
	let modalName = $state('');
	let modalPermissions = $state<string[]>(ALL_PERMISSIONS.map((p) => p.value));
	let creating = $state(false);
	let createError = $state('');

	// Newly created key display
	let createdKey = $state<string | null>(null);
	let copied = $state(false);

	let atLimit = $derived(keys.length >= MAX_KEYS);

	onMount(async () => {
		if (!authStore.isAuthenticated()) {
			goto('/account/login');
			return;
		}
		await loadKeys();
	});

	async function loadKeys() {
		loading = true;
		error = '';
		try {
			const res = await apiKeysApi.list();
			keys = res.data ?? [];
		} catch {
			error = $t('common.error');
		} finally {
			loading = false;
		}
	}

	function openModal() {
		modalName = '';
		modalPermissions = ALL_PERMISSIONS.map((p) => p.value);
		createError = '';
		showModal = true;
	}

	function closeModal() {
		showModal = false;
		createError = '';
	}

	function togglePermission(value: string) {
		if (modalPermissions.includes(value)) {
			modalPermissions = modalPermissions.filter((p) => p !== value);
		} else {
			modalPermissions = [...modalPermissions, value];
		}
	}

	async function handleCreate() {
		if (!modalName.trim()) return;
		creating = true;
		createError = '';
		try {
			const res = await apiKeysApi.create(modalName.trim(), modalPermissions);
			if (res.data) {
				const newKey = res.data as CreateKeyResponse;
				createdKey = newKey.key;
				keys = [...keys, newKey];
				showModal = false;
			} else {
				createError = $t('common.error');
			}
		} catch {
			createError = $t('common.error');
		} finally {
			creating = false;
		}
	}

	async function handleRevoke(id: string) {
		if (!window.confirm($t('apiKeys.revokeConfirm'))) return;
		try {
			await apiKeysApi.revoke(id);
			keys = keys.filter((k) => k.id !== id);
		} catch {
			// silently fail — user can refresh
		}
	}

	async function copyKey() {
		if (!createdKey) return;
		try {
			await navigator.clipboard.writeText(createdKey);
			copied = true;
			setTimeout(() => (copied = false), 2000);
		} catch {
			// clipboard not available
		}
	}

	function dismissCreatedKey() {
		createdKey = null;
		copied = false;
	}
</script>

<svelte:head>
	<title>{$t('apiKeys.pageTitle')}</title>
</svelte:head>

<div class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
	<!-- Page header -->
	<div class="flex items-start justify-between mb-2">
		<div>
			<h1 class="text-2xl font-bold text-gray-900">{$t('apiKeys.title')}</h1>
			<p class="text-sm text-gray-500 mt-1">{$t('apiKeys.description')}</p>
		</div>
		{#if !atLimit}
			<button onclick={openModal} class="btn btn-primary shrink-0 mt-1">
				{$t('apiKeys.createButton')}
			</button>
		{/if}
	</div>

	<!-- Limit hint -->
	{#if atLimit}
		<p class="text-sm text-amber-600 bg-amber-50 border border-amber-200 rounded-lg px-4 py-2 mt-4 mb-6">
			{$t('apiKeys.limitReached')}
		</p>
	{:else}
		<div class="mb-6"></div>
	{/if}

	<!-- Newly created key banner -->
	{#if createdKey}
		<div class="card p-5 mb-6 border-2 border-emerald-400 bg-emerald-50">
			<div class="flex items-start justify-between gap-4 mb-3">
				<div>
					<p class="font-semibold text-emerald-900">{$t('apiKeys.keyCreated')}</p>
					<p class="text-sm text-emerald-700 mt-0.5">{$t('apiKeys.keyCreatedHint')}</p>
				</div>
				<button
					onclick={dismissCreatedKey}
					class="text-emerald-500 hover:text-emerald-700 transition-colors shrink-0"
					aria-label="Dismiss"
				>
					<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
						<path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" />
					</svg>
				</button>
			</div>
			<div class="flex items-center gap-2">
				<code class="flex-1 text-sm font-mono bg-white border border-emerald-200 rounded-lg px-3 py-2 text-gray-800 break-all">
					{createdKey}
				</code>
				<button
					onclick={copyKey}
					class="btn btn-secondary shrink-0 text-sm {copied ? 'text-emerald-700 border-emerald-400' : ''}"
				>
					{#if copied}
						<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 mr-1.5 inline" viewBox="0 0 20 20" fill="currentColor">
							<path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
						</svg>
						{$t('apiKeys.copied')}
					{:else}
						<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 mr-1.5 inline" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
						</svg>
						Copy
					{/if}
				</button>
			</div>
		</div>
	{/if}

	<!-- Loading skeleton -->
	{#if loading}
		<div class="animate-pulse space-y-4">
			{#each Array(2) as _}
				<div class="h-24 bg-gray-100 rounded-xl"></div>
			{/each}
		</div>

	<!-- Error -->
	{:else if error}
		<p class="text-red-600">{error}</p>

	<!-- Empty state -->
	{:else if keys.length === 0}
		<div class="text-center py-20">
			<div class="inline-flex items-center justify-center w-14 h-14 rounded-full bg-gray-100 mb-4">
				<svg xmlns="http://www.w3.org/2000/svg" class="h-7 w-7 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
				</svg>
			</div>
			<p class="font-medium text-gray-700">{$t('apiKeys.empty')}</p>
			<p class="text-sm text-gray-400 mt-1">{$t('apiKeys.emptyHint')}</p>
			<button onclick={openModal} class="btn btn-primary mt-6">
				{$t('apiKeys.createButton')}
			</button>
		</div>

	<!-- Key list -->
	{:else}
		<div class="space-y-4">
			{#each keys as key (key.id)}
				<div class="card p-5">
					<div class="flex items-start justify-between gap-4">
						<div class="min-w-0">
							<p class="font-semibold text-gray-900 truncate">{key.name}</p>
							<div class="flex flex-wrap gap-x-4 gap-y-1 mt-1 text-xs text-gray-500">
								<span>
									{$t('apiKeys.createdAt')}: {$fmt.date(key.created_at)}
								</span>
								<span>
									{$t('apiKeys.lastUsed')}:
									{key.last_used_at ? $fmt.date(key.last_used_at) : $t('apiKeys.lastUsedNever')}
								</span>
							</div>
							{#if key.permissions && key.permissions.length > 0}
								<div class="flex flex-wrap gap-1.5 mt-2.5">
									{#each key.permissions as perm}
										<span class="badge badge-gray text-xs font-mono">{perm}</span>
									{/each}
								</div>
							{/if}
						</div>
						<button
							onclick={() => handleRevoke(key.id)}
							class="btn btn-secondary text-sm text-red-600 border-red-200 hover:bg-red-50 shrink-0"
						>
							{$t('apiKeys.revoke')}
						</button>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

<!-- Create modal -->
{#if showModal}
	<!-- Backdrop -->
	<button
		class="fixed inset-0 z-40 w-full h-full bg-black/40 backdrop-blur-sm cursor-default"
		onclick={closeModal}
		aria-label="Close dialog"
		tabindex="-1"
	></button>

	<!-- Dialog -->
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-4"
		role="dialog"
		aria-modal="true"
	>
		<div class="card w-full max-w-md p-6 shadow-2xl">
			<div class="flex items-center justify-between mb-5">
				<h2 class="text-lg font-bold text-gray-900">{$t('apiKeys.createButton')}</h2>
				<button
					onclick={closeModal}
					class="text-gray-400 hover:text-gray-600 transition-colors"
					aria-label="Close"
				>
					<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
						<path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" />
					</svg>
				</button>
			</div>

			<!-- Name field -->
			<div class="mb-5">
				<label for="key-name" class="block text-sm font-medium text-gray-700 mb-1.5">
					{$t('apiKeys.name')}
				</label>
				<input
					id="key-name"
					type="text"
					bind:value={modalName}
					placeholder={$t('apiKeys.namePlaceholder')}
					class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm text-gray-900 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
				/>
			</div>

			<!-- Permissions -->
			<div class="mb-6">
				<p class="text-sm font-medium text-gray-700 mb-2">{$t('apiKeys.permissions')}</p>
				<div class="space-y-2">
					{#each ALL_PERMISSIONS as perm}
						<label class="flex items-center gap-2.5 cursor-pointer group">
							<input
								type="checkbox"
								checked={modalPermissions.includes(perm.value)}
								onchange={() => togglePermission(perm.value)}
								class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
							/>
							<span class="text-sm text-gray-700 group-hover:text-gray-900">
								{$t(perm.labelKey)}
								<span class="ml-1 text-xs text-gray-400 font-mono">{perm.value}</span>
							</span>
						</label>
					{/each}
				</div>
			</div>

			{#if createError}
				<p class="text-sm text-red-600 mb-4">{createError}</p>
			{/if}

			<div class="flex gap-3 justify-end">
				<button onclick={closeModal} class="btn btn-secondary">
					{$t('apiKeys.cancel')}
				</button>
				<button
					onclick={handleCreate}
					disabled={creating || !modalName.trim()}
					class="btn btn-primary disabled:opacity-50 disabled:cursor-not-allowed"
				>
					{creating ? '...' : $t('apiKeys.create')}
				</button>
			</div>
		</div>
	</div>
{/if}
