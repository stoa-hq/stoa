<script lang="ts">
  import { onMount } from 'svelte';
  import { t } from 'svelte-i18n';
  import { apiKeysApi } from '$lib/api/api-keys';
  import { notifications } from '$lib/stores/notifications';
  import { authStore } from '$lib/stores/auth';
  import Modal from '$lib/components/Modal.svelte';
  import ConfirmModal from '$lib/components/ConfirmModal.svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';
  import type { APIKey, APIKeyCreateResponse } from '$lib/types';

  // ── State ──────────────────────────────────────────────────────────────────

  let items = $state<APIKey[]>([]);
  let loading = $state(true);
  let showCreateModal = $state(false);
  let showSuccessModal = $state(false);
  let showConfirm = $state(false);
  let revokeId = $state<string | null>(null);
  let submitting = $state(false);
  let copied = $state(false);
  let showAll = $state(false);

  let createdKey = $state<APIKeyCreateResponse | null>(null);
  let form = $state({ name: '', permissions: [] as string[] });

  // ── Role detection from JWT ────────────────────────────────────────────────

  let userRole = $state('');
  $effect(() => {
    const unsubscribe = authStore.subscribe((auth) => {
      userRole = auth.user?.role ?? '';
    });
    return unsubscribe;
  });

  const isSuperAdmin = $derived(userRole === 'super_admin');

  // ── Permission groups ──────────────────────────────────────────────────────

  const permissionGroups = [
    {
      label: 'Products',
      perms: ['products.create', 'products.read', 'products.update', 'products.delete']
    },
    {
      label: 'Categories',
      perms: ['categories.create', 'categories.read', 'categories.update', 'categories.delete']
    },
    {
      label: 'Customers',
      perms: ['customers.create', 'customers.read', 'customers.update', 'customers.delete']
    },
    {
      label: 'Orders',
      perms: ['orders.create', 'orders.read', 'orders.update', 'orders.delete']
    },
    {
      label: 'Media',
      perms: ['media.create', 'media.read', 'media.delete']
    },
    {
      label: 'Discounts',
      perms: ['discounts.create', 'discounts.read', 'discounts.update', 'discounts.delete']
    },
    {
      label: 'Shipping',
      perms: ['shipping.create', 'shipping.read', 'shipping.update', 'shipping.delete']
    },
    {
      label: 'Payment',
      perms: ['payment.create', 'payment.read', 'payment.update', 'payment.delete']
    },
    {
      label: 'Tax',
      perms: ['tax.create', 'tax.read', 'tax.update', 'tax.delete']
    },
    {
      label: 'Settings',
      perms: ['settings.read', 'settings.update']
    },
    {
      label: 'Plugins',
      perms: ['plugins.manage']
    },
    {
      label: 'Audit',
      perms: ['audit.read']
    },
    {
      label: 'API Keys',
      perms: ['api_keys.manage']
    }
  ];

  const allPermissions = permissionGroups.flatMap((g) => g.perms);

  const allSelected = $derived(
    allPermissions.length > 0 && allPermissions.every((p) => form.permissions.includes(p))
  );

  // ── Data loading ───────────────────────────────────────────────────────────

  async function load() {
    loading = true;
    try {
      const res = await apiKeysApi.list(showAll && isSuperAdmin ? true : undefined);
      items = res.data ?? [];
    } catch {
      notifications.error($t('apiKeys.loadFailed'));
    } finally {
      loading = false;
    }
  }

  onMount(load);

  // ── Create ─────────────────────────────────────────────────────────────────

  function openCreateModal() {
    form = { name: '', permissions: [] };
    showCreateModal = true;
  }

  function togglePermission(perm: string) {
    if (form.permissions.includes(perm)) {
      form.permissions = form.permissions.filter((p) => p !== perm);
    } else {
      form.permissions = [...form.permissions, perm];
    }
  }

  function toggleAll() {
    if (allSelected) {
      form.permissions = [];
    } else {
      form.permissions = [...allPermissions];
    }
  }

  function setMcpFullAccess() {
    form.permissions = [...allPermissions];
  }

  async function handleCreate(e: SubmitEvent) {
    e.preventDefault();
    if (!form.name.trim()) return;
    submitting = true;
    try {
      const res = await apiKeysApi.create({ name: form.name.trim(), permissions: form.permissions });
      createdKey = res.data;
      showCreateModal = false;
      showSuccessModal = true;
      notifications.success($t('apiKeys.created'));
      load();
    } catch {
      notifications.error($t('apiKeys.createFailed'));
    } finally {
      submitting = false;
    }
  }

  // ── Copy ───────────────────────────────────────────────────────────────────

  async function copyKey() {
    if (!createdKey?.key) return;
    try {
      await navigator.clipboard.writeText(createdKey.key);
      copied = true;
      setTimeout(() => (copied = false), 2000);
    } catch {
      // fallback: select the input
    }
  }

  // ── Revoke ─────────────────────────────────────────────────────────────────

  function confirmRevoke(id: string, e: MouseEvent) {
    e.stopPropagation();
    revokeId = id;
    showConfirm = true;
  }

  async function doRevoke() {
    if (!revokeId) return;
    try {
      await apiKeysApi.revoke(revokeId);
      notifications.success($t('apiKeys.revoked'));
      showConfirm = false;
      revokeId = null;
      load();
    } catch {
      notifications.error($t('common.deleteFailed'));
    }
  }

  // ── Formatting ─────────────────────────────────────────────────────────────

  function formatDate(iso: string | undefined): string {
    if (!iso) return $t('apiKeys.never');
    return new Intl.DateTimeFormat(undefined, {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    }).format(new Date(iso));
  }
</script>

<!-- ── Header ──────────────────────────────────────────────────────────────── -->
<div class="flex items-center justify-between mb-6">
  <h1 class="text-2xl font-bold text-[var(--text)]">{$t('apiKeys.title')}</h1>
  <button class="btn btn-primary" onclick={openCreateModal}>
    {$t('apiKeys.newKey')}
  </button>
</div>

<!-- ── Super Admin Toggle ──────────────────────────────────────────────────── -->
{#if isSuperAdmin}
  <div class="mb-4 flex items-center gap-2">
    <input
      id="show-all"
      type="checkbox"
      class="w-4 h-4 rounded border-[var(--card-border)] text-primary-600 focus:ring-primary-500"
      checked={showAll}
      onchange={(e) => { showAll = (e.currentTarget as HTMLInputElement).checked; load(); }}
    />
    <label for="show-all" class="text-sm text-[var(--text-muted)] cursor-pointer select-none">
      {$t('apiKeys.showAll')}
    </label>
  </div>
{/if}

<!-- ── Table ───────────────────────────────────────────────────────────────── -->
<div class="card p-6">
  {#if loading}
    <div class="space-y-3">
      {#each Array(4) as _}
        <Skeleton height="h-10" />
      {/each}
    </div>
  {:else}
    <div class="overflow-x-auto">
      <table class="min-w-full divide-y divide-[var(--card-border)]">
        <thead>
          <tr>
            <th class="table-header">{$t('common.name')}</th>
            <th class="table-header">{$t('apiKeys.permissions')}</th>
            <th class="table-header">{$t('common.status')}</th>
            <th class="table-header">{$t('apiKeys.lastUsed')}</th>
            <th class="table-header">{$t('common.createdAt')}</th>
            <th class="table-header">{$t('common.actions')}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-[var(--card-border)]">
          {#each items as item}
            <tr class="table-row">
              <td class="table-cell font-medium text-[var(--text)]">{item.name}</td>
              <td class="table-cell">
                <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300">
                  {item.permissions?.length ?? 0} {$t('apiKeys.permissions').toLowerCase()}
                </span>
              </td>
              <td class="table-cell">
                {#if item.active}
                  <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300">
                    {$t('apiKeys.active')}
                  </span>
                {:else}
                  <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-600 dark:bg-gray-700/50 dark:text-gray-400">
                    {$t('apiKeys.revokedStatus')}
                  </span>
                {/if}
              </td>
              <td class="table-cell text-[var(--text-muted)] text-sm">
                {formatDate(item.last_used_at)}
              </td>
              <td class="table-cell text-[var(--text-muted)] text-sm">
                {formatDate(item.created_at)}
              </td>
              <td class="table-cell text-right">
                {#if item.active}
                  <button
                    class="btn btn-danger btn-sm"
                    onclick={(e) => confirmRevoke(item.id, e)}
                  >
                    {$t('apiKeys.revoke')}
                  </button>
                {/if}
              </td>
            </tr>
          {/each}
          {#if items.length === 0}
            <tr>
              <td colspan="6" class="table-cell text-center text-[var(--text-muted)] py-8">
                {$t('apiKeys.noKeys')}
              </td>
            </tr>
          {/if}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<!-- ── Create Modal ────────────────────────────────────────────────────────── -->
<Modal
  open={showCreateModal}
  title={$t('apiKeys.newKey')}
  onClose={() => (showCreateModal = false)}
>
  <form onsubmit={handleCreate} class="space-y-5">
    <!-- Name -->
    <div>
      <label class="label" for="key-name">{$t('apiKeys.name')} *</label>
      <input
        id="key-name"
        class="input"
        type="text"
        bind:value={form.name}
        required
        placeholder={$t('apiKeys.namePlaceholder')}
        autocomplete="off"
      />
    </div>

    <!-- Permission controls -->
    <div>
      <div class="flex items-center justify-between mb-3">
        <span class="label mb-0">{$t('apiKeys.permissions')}</span>
        <div class="flex gap-2">
          <button
            type="button"
            class="btn btn-secondary btn-sm"
            onclick={setMcpFullAccess}
          >
            {$t('apiKeys.mcpFullAccess')}
          </button>
          <button
            type="button"
            class="btn btn-secondary btn-sm"
            onclick={toggleAll}
          >
            {allSelected ? $t('apiKeys.deselectAll') : $t('apiKeys.selectAll')}
          </button>
        </div>
      </div>

      <!-- Permission groups grid -->
      <div class="space-y-3 max-h-72 overflow-y-auto pr-1">
        {#each permissionGroups as group}
          <div class="rounded-lg border border-[var(--card-border)] p-3">
            <p class="text-xs font-semibold text-[var(--text-muted)] uppercase tracking-wider mb-2">
              {group.label}
            </p>
            <div class="flex flex-wrap gap-x-4 gap-y-1.5">
              {#each group.perms as perm}
                {@const checked = form.permissions.includes(perm)}
                <label class="flex items-center gap-1.5 text-sm text-[var(--text)] cursor-pointer">
                  <input
                    type="checkbox"
                    class="w-3.5 h-3.5 rounded border-[var(--card-border)] text-primary-600 focus:ring-primary-500"
                    checked={checked}
                    onchange={() => togglePermission(perm)}
                  />
                  <span class="font-mono text-xs">{perm.split('.')[1]}</span>
                </label>
              {/each}
            </div>
          </div>
        {/each}
      </div>
    </div>

    <!-- Footer buttons -->
    <div class="flex gap-3 pt-2">
      <button type="submit" class="btn btn-primary" disabled={submitting || !form.name.trim()}>
        {submitting ? $t('common.creating') : $t('common.create')}
      </button>
      <button
        type="button"
        class="btn btn-secondary"
        onclick={() => (showCreateModal = false)}
      >
        {$t('common.cancel')}
      </button>
    </div>
  </form>
</Modal>

<!-- ── Success Modal ───────────────────────────────────────────────────────── -->
<Modal
  open={showSuccessModal}
  title={$t('apiKeys.keyCreatedTitle')}
  onClose={() => { showSuccessModal = false; createdKey = null; }}
>
  {#if createdKey}
    <div class="space-y-4">
      <!-- Warning banner -->
      <div class="flex items-start gap-3 rounded-lg bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-700/40 px-4 py-3">
        <svg class="w-5 h-5 text-amber-600 dark:text-amber-400 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z" />
        </svg>
        <p class="text-sm text-amber-800 dark:text-amber-300 font-medium">
          {$t('apiKeys.keyCreatedWarning')}
        </p>
      </div>

      <!-- Key display -->
      <div>
        <label class="label" for="created-key">{$t('apiKeys.name')}</label>
        <p class="text-sm font-medium text-[var(--text)] mb-2">{createdKey.name}</p>
        <div class="flex gap-2">
          <input
            id="created-key"
            type="text"
            class="input font-mono text-sm flex-1"
            value={createdKey.key}
            readonly
            onclick={(e) => (e.currentTarget as HTMLInputElement).select()}
          />
          <button
            type="button"
            class="btn btn-secondary shrink-0"
            onclick={copyKey}
          >
            {copied ? $t('apiKeys.copied') : $t('apiKeys.copyKey')}
          </button>
        </div>
      </div>

      <!-- MCP hint -->
      <div class="rounded-lg bg-[var(--surface)] border border-[var(--card-border)] px-4 py-3">
        <p class="text-xs font-semibold text-[var(--text-muted)] uppercase tracking-wider mb-1.5">
          {$t('apiKeys.mcpHint')}
        </p>
        <code class="text-xs font-mono text-[var(--text)] break-all">
          STOA_MCP_API_KEY={createdKey.key}
        </code>
      </div>

      <!-- Close -->
      <div class="flex justify-end pt-1">
        <button
          type="button"
          class="btn btn-primary"
          onclick={() => { showSuccessModal = false; createdKey = null; }}
        >
          {$t('common.close')}
        </button>
      </div>
    </div>
  {/if}
</Modal>

<!-- ── Confirm Revoke Modal ────────────────────────────────────────────────── -->
<ConfirmModal
  open={showConfirm}
  title={$t('apiKeys.revokeTitle')}
  message={$t('apiKeys.revokeMessage')}
  confirmLabel={$t('apiKeys.revoke')}
  danger={true}
  onConfirm={doRevoke}
  onCancel={() => { showConfirm = false; revokeId = null; }}
/>
