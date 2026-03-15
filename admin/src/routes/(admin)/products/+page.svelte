<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { productsApi } from '$lib/api/products';
  import { notifications } from '$lib/stores/notifications';
  import Pagination from '$lib/components/Pagination.svelte';
  import ConfirmModal from '$lib/components/ConfirmModal.svelte';
  import ProductImportModal from '$lib/components/ProductImportModal.svelte';
  import SearchBar from '$lib/components/SearchBar.svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';
  import { t, locale } from 'svelte-i18n';
  import { fmt } from '$lib/i18n/formatters';
  import { tr } from '$lib/i18n/entity';

  let items = $state<any[]>([]);
  let meta = $state<any>(null);
  let currentPage = $state(1);
  let limit = $state(25);
  let loading = $state(true);
  let search = $state('');
  let deleteId = $state<string | null>(null);
  let showConfirm = $state(false);
  let showImport = $state(false);

  function openImport() { showImport = true; }

  function handleImported() {
    notifications.success($t('products.importSuccess'));
    load();
  }

  async function load() {
    loading = true;
    try {
      const res = await productsApi.list({ page: currentPage, limit, search: search || undefined });
      items = res.data?.items ?? [];
      meta = res.meta ?? null;
    } catch (e) {
      notifications.error($t('products.loadFailed'));
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
      await productsApi.delete(deleteId);
      notifications.success($t('products.deleted'));
      showConfirm = false;
      deleteId = null;
      load();
    } catch (e) {
      notifications.error($t('common.deleteFailed'));
    }
  }
</script>

<div class="flex items-center justify-between mb-6">
  <h1 class="text-2xl font-bold text-[var(--text)]">{$t('products.title')}</h1>
  <div class="flex gap-2">
    <button class="btn btn-secondary" onclick={openImport}>{$t('common.import')}</button>
    <a href="{base}/products/new" class="btn btn-primary">{$t('common.new')}</a>
  </div>
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
            <th class="table-header">{$t('products.sku')}</th>
            <th class="table-header">{$t('common.name')}</th>
            <th class="table-header">{$t('products.priceGross')}</th>
            <th class="table-header">{$t('products.stock')}</th>
            <th class="table-header">{$t('common.active')}</th>
            <th class="table-header">{$t('common.createdAt')}</th>
            <th class="table-header"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-[var(--card-border)]">
          {#each items as item}
            <tr
              class="table-row cursor-pointer"
              onclick={() => goto(`${base}/products/${item.id}`)}
            >
              <td class="table-cell text-[var(--text-muted)] font-mono text-xs">{item.sku}</td>
              <td class="table-cell font-medium text-[var(--text)]">{tr(item.translations, 'name', $locale) || item.sku}</td>
              <td class="table-cell text-[var(--text-muted)] tabular-nums">{$fmt.price(item.price_gross)}</td>
              <td class="table-cell text-[var(--text-muted)] tabular-nums">{item.stock ?? 0}</td>
              <td class="table-cell">
                {#if item.active}
                  <span class="badge badge-green">{$t('common.active')}</span>
                {:else}
                  <span class="badge badge-gray">{$t('common.inactive')}</span>
                {/if}
              </td>
              <td class="table-cell text-[var(--text-muted)]">{$fmt.date(item.created_at)}</td>
              <td class="table-cell text-right">
                <button
                  class="btn btn-danger btn-sm"
                  onclick={(e) => confirmDelete(item.id, e)}
                >
                  {$t('common.delete')}
                </button>
              </td>
            </tr>
          {/each}
          {#if items.length === 0}
            <tr>
              <td colspan="7" class="table-cell text-center text-[var(--text-muted)] py-6">{$t('products.noProductsFound')}</td>
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
          onclick={() => goto(`${base}/products/${item.id}`)}
          onkeydown={(e) => e.key === 'Enter' && goto(`${base}/products/${item.id}`)}
        >
          <div class="flex items-center justify-between mb-1">
            <span class="font-medium text-sm text-[var(--text)]">{tr(item.translations, 'name', $locale) || item.sku}</span>
            {#if item.active}
              <span class="badge badge-green">{$t('common.active')}</span>
            {:else}
              <span class="badge badge-gray">{$t('common.inactive')}</span>
            {/if}
          </div>
          <div class="flex items-center justify-between text-xs text-[var(--text-muted)]">
            <span class="tabular-nums">{$fmt.price(item.price_gross)}</span>
            <span>{$t('products.stock')}: {item.stock ?? 0}</span>
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
  title={$t('products.deleteTitle')}
  message={$t('products.deleteMessage')}
  onConfirm={doDelete}
  onCancel={() => { showConfirm = false; deleteId = null; }}
/>

<ProductImportModal
  open={showImport}
  onClose={() => { showImport = false; }}
  onImported={handleImported}
/>
