<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { productsApi } from '$lib/api/products';
  import { notifications } from '$lib/stores/notifications';
  import Pagination from '$lib/components/Pagination.svelte';
  import ConfirmModal from '$lib/components/ConfirmModal.svelte';
  import ProductImportModal from '$lib/components/ProductImportModal.svelte';
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

  let searchTimeout: ReturnType<typeof setTimeout>;
  function handleSearch() {
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => {
      currentPage = 1;
      load();
    }, 400);
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
  <h1 class="text-2xl font-bold text-gray-900">{$t('products.title')}</h1>
  <div class="flex gap-2">
    <button class="btn btn-secondary" onclick={openImport}>{$t('common.import')}</button>
    <a href="{base}/products/new" class="btn btn-primary">{$t('common.new')}</a>
  </div>
</div>

<div class="card p-6">
  <div class="mb-4">
    <input
      class="input max-w-xs"
      type="search"
      placeholder={$t('common.search')}
      bind:value={search}
      oninput={handleSearch}
    />
  </div>

  {#if loading}
    <div class="flex items-center justify-center h-32">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
    </div>
  {:else}
    <div class="overflow-x-auto">
      <table class="min-w-full divide-y divide-gray-200">
        <thead>
          <tr>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{$t('products.sku')}</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{$t('common.name')}</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{$t('products.priceGross')}</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{$t('products.stock')}</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{$t('common.active')}</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{$t('common.createdAt')}</th>
            <th class="px-4 py-3"></th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          {#each items as item}
            <tr
              class="hover:bg-gray-50 cursor-pointer"
              onclick={() => goto(`${base}/products/${item.id}`)}
            >
              <td class="px-4 py-3 text-sm text-gray-600">{item.sku}</td>
              <td class="px-4 py-3 text-sm font-medium text-gray-900">{tr(item.translations, 'name', $locale) || item.sku}</td>
              <td class="px-4 py-3 text-sm text-gray-700">{$fmt.price(item.price_gross)}</td>
              <td class="px-4 py-3 text-sm text-gray-700">{item.stock ?? 0}</td>
              <td class="px-4 py-3 text-sm">
                {#if item.active}
                  <span class="badge badge-green">{$t('common.active')}</span>
                {:else}
                  <span class="badge badge-gray">{$t('common.inactive')}</span>
                {/if}
              </td>
              <td class="px-4 py-3 text-sm text-gray-500">{$fmt.date(item.created_at)}</td>
              <td class="px-4 py-3 text-right">
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
              <td colspan="7" class="px-4 py-6 text-center text-gray-400 text-sm">{$t('products.noProductsFound')}</td>
            </tr>
          {/if}
        </tbody>
      </table>
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
