<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { t } from 'svelte-i18n';
  import { customersApi } from '$lib/api/customers';
  import { notifications } from '$lib/stores/notifications';
  import { fmt } from '$lib/i18n/formatters';
  import Pagination from '$lib/components/Pagination.svelte';
  import ConfirmModal from '$lib/components/ConfirmModal.svelte';
  import SearchBar from '$lib/components/SearchBar.svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';

  let items = $state<any[]>([]);
  let meta = $state<any>(null);
  let currentPage = $state(1);
  let limit = $state(25);
  let loading = $state(true);
  let deleteId = $state<string | null>(null);
  let showConfirm = $state(false);
  let search = $state('');

  async function load() {
    loading = true;
    try {
      const params: any = { page: currentPage, limit };
      if (search) params.search = search;
      const res = await customersApi.list(params);
      items = res.data ?? [];
      meta = res.meta ?? null;
    } catch (e) {
      notifications.error($t('customers.loadFailed'));
    } finally {
      loading = false;
    }
  }

  onMount(load);

  function handlePageChange(p: number) {
    currentPage = p;
    load();
  }

  function handleSearch(value: string) {
    search = value;
    currentPage = 1;
    load();
  }

  function confirmDelete(id: string, e: MouseEvent) {
    e.stopPropagation();
    deleteId = id;
    showConfirm = true;
  }

  async function doDelete() {
    if (!deleteId) return;
    try {
      await customersApi.delete(deleteId);
      notifications.success($t('customers.deleted'));
      showConfirm = false;
      deleteId = null;
      load();
    } catch (e) {
      notifications.error($t('common.deleteFailed'));
    }
  }
</script>

<div class="flex items-center justify-between mb-6">
  <h1 class="text-2xl font-bold text-[var(--text)]">{$t('customers.title')}</h1>
</div>

<div class="card p-6">
  <div class="mb-4">
    <SearchBar value={search} onSearch={handleSearch} />
  </div>

  {#if loading}
    <div class="space-y-3">
      {#each Array(5) as _}
        <Skeleton height="h-12" />
      {/each}
    </div>
  {:else}
    <div class="hidden sm:block overflow-x-auto">
      <table class="min-w-full divide-y divide-[var(--card-border)]">
        <thead>
          <tr>
            <th class="table-header">{$t('common.email')}</th>
            <th class="table-header">{$t('customers.firstName')}</th>
            <th class="table-header">{$t('customers.lastName')}</th>
            <th class="table-header">{$t('common.active')}</th>
            <th class="table-header">{$t('common.createdAt')}</th>
            <th class="table-header"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-[var(--card-border)]">
          {#each items as item}
            <tr class="table-row cursor-pointer" onclick={() => goto(`${base}/customers/${item.id}`)}>
              <td class="table-cell font-medium text-[var(--text)]">{item.email}</td>
              <td class="table-cell text-[var(--text-muted)]">{item.first_name ?? '—'}</td>
              <td class="table-cell text-[var(--text-muted)]">{item.last_name ?? '—'}</td>
              <td class="table-cell">
                {#if item.active}
                  <span class="badge badge-green">{$t('common.active')}</span>
                {:else}
                  <span class="badge badge-gray">{$t('common.inactive')}</span>
                {/if}
              </td>
              <td class="table-cell text-[var(--text-muted)]">{$fmt.date(item.created_at)}</td>
              <td class="table-cell text-right">
                <button class="btn btn-danger btn-sm opacity-0 group-hover:opacity-100" onclick={(e) => confirmDelete(item.id, e)}>{$t('common.delete')}</button>
              </td>
            </tr>
          {/each}
          {#if items.length === 0}
            <tr>
              <td colspan="6" class="table-cell text-center text-[var(--text-muted)] py-6">{$t('customers.noCustomers')}</td>
            </tr>
          {/if}
        </tbody>
      </table>
    </div>
    <!-- Mobile Cards -->
    <div class="sm:hidden space-y-3">
      {#each items as item}
        <div
          class="p-3 rounded-lg bg-[var(--surface)] border border-[var(--card-border)] cursor-pointer hover:bg-gray-50 dark:hover:bg-white/5 transition-colors"
          role="button" tabindex="0"
          onclick={() => goto(`${base}/customers/${item.id}`)}
          onkeydown={(e) => e.key === 'Enter' && goto(`${base}/customers/${item.id}`)}
        >
          <div class="flex items-center justify-between mb-1">
            <span class="font-medium text-sm text-[var(--text)]">{item.email}</span>
            {#if item.active}
              <span class="badge badge-green">{$t('common.active')}</span>
            {:else}
              <span class="badge badge-gray">{$t('common.inactive')}</span>
            {/if}
          </div>
          <div class="flex items-center justify-between text-xs text-[var(--text-muted)]">
            <span>{item.first_name ?? ''} {item.last_name ?? ''}</span>
            <span>{$fmt.date(item.created_at)}</span>
          </div>
        </div>
      {/each}
    </div>

    {#if meta}
      <div class="mt-4">
        <Pagination
          currentPage={currentPage}
          totalPages={Math.ceil(meta.total / limit)}
          onPageChange={handlePageChange}
        />
      </div>
    {/if}
  {/if}
</div>

<ConfirmModal
  open={showConfirm}
  title={$t('customers.deleteTitle')}
  message={$t('customers.deleteMessage')}
  onConfirm={doDelete}
  onCancel={() => { showConfirm = false; deleteId = null; }}
/>
