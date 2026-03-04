<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { productsApi } from '$lib/api/products';
  import { notifications } from '$lib/stores/notifications';
  import { formatPrice, formatDate } from '$lib/utils';
  import Pagination from '$lib/components/Pagination.svelte';
  import ConfirmModal from '$lib/components/ConfirmModal.svelte';

  let items = $state<any[]>([]);
  let meta = $state<any>(null);
  let currentPage = $state(1);
  let limit = $state(25);
  let loading = $state(true);
  let search = $state('');
  let deleteId = $state<string | null>(null);
  let showConfirm = $state(false);

  async function load() {
    loading = true;
    try {
      const res = await productsApi.list({ page: currentPage, limit, search: search || undefined });
      items = res.data?.items ?? [];
      meta = res.meta ?? null;
    } catch (e) {
      notifications.error('Produkte konnten nicht geladen werden.');
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
      notifications.success('Produkt gelöscht.');
      showConfirm = false;
      deleteId = null;
      load();
    } catch (e) {
      notifications.error('Löschen fehlgeschlagen.');
    }
  }
</script>

<div class="flex items-center justify-between mb-6">
  <h1 class="text-2xl font-bold text-gray-900">Produkte</h1>
  <a href="{base}/products/new" class="btn btn-primary">+ Neu</a>
</div>

<div class="card p-6">
  <div class="mb-4">
    <input
      class="input max-w-xs"
      type="search"
      placeholder="Suchen..."
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
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">SKU</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Name</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Preis (brutto)</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Lager</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Aktiv</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Erstellt</th>
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
              <td class="px-4 py-3 text-sm font-medium text-gray-900">{item.translations?.[0]?.name ?? item.sku}</td>
              <td class="px-4 py-3 text-sm text-gray-700">{formatPrice(item.price_gross)}</td>
              <td class="px-4 py-3 text-sm text-gray-700">{item.stock ?? 0}</td>
              <td class="px-4 py-3 text-sm">
                {#if item.active}
                  <span class="badge badge-green">Aktiv</span>
                {:else}
                  <span class="badge badge-gray">Inaktiv</span>
                {/if}
              </td>
              <td class="px-4 py-3 text-sm text-gray-500">{formatDate(item.created_at)}</td>
              <td class="px-4 py-3 text-right">
                <button
                  class="btn btn-danger btn-sm"
                  onclick={(e) => confirmDelete(item.id, e)}
                >
                  Löschen
                </button>
              </td>
            </tr>
          {/each}
          {#if items.length === 0}
            <tr>
              <td colspan="7" class="px-4 py-6 text-center text-gray-400 text-sm">Keine Produkte gefunden.</td>
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
  title="Produkt löschen"
  message="Soll dieses Produkt wirklich gelöscht werden? Diese Aktion kann nicht rückgängig gemacht werden."
  onConfirm={doDelete}
  onCancel={() => { showConfirm = false; deleteId = null; }}
/>
