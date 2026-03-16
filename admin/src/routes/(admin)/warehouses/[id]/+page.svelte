<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { page } from '$app/stores';
  import { t, locale } from 'svelte-i18n';
  import { warehousesApi } from '$lib/api/warehouses';
  import { productsApi } from '$lib/api/products';
  import { notifications } from '$lib/stores/notifications';
  import { tr } from '$lib/i18n/entity';
  import ConfirmModal from '$lib/components/ConfirmModal.svelte';
  import Modal from '$lib/components/Modal.svelte';

  let id = $derived($page.params.id as string);
  let loading = $state(true);
  let submitting = $state(false);
  let showDeleteConfirm = $state(false);

  let form = $state({
    name: '',
    code: '',
    active: true,
    allow_negative_stock: false,
    priority: 0,
    address_line1: '',
    address_line2: '',
    city: '',
    state: '',
    postal_code: '',
    country: '',
  });

  let stockItems = $state<any[]>([]);
  let stockLoading = $state(true);

  // Stock editing
  let editingStock = $state(false);
  let stockEdits = $state<Record<string, number>>({});
  let savingStock = $state(false);

  // Add product modal — search results are flattened: products + variants as separate entries
  interface SearchResult {
    product_id: string;
    variant_id?: string;
    label: string;      // display name
    sku: string;
    isVariant: boolean;
  }

  let showAddProduct = $state(false);
  let productSearch = $state('');
  let searchResults = $state<SearchResult[]>([]);
  let searchLoading = $state(false);
  let selectedResult = $state<SearchResult | null>(null);
  let addQuantity = $state(0);
  let addingProduct = $state(false);
  let searchTimeout: ReturnType<typeof setTimeout> | undefined;

  onMount(async () => {
    try {
      const [res, stockRes] = await Promise.all([
        warehousesApi.get(id),
        warehousesApi.getStock(id),
      ]);
      const wh = res.data;
      form = {
        name: wh.name,
        code: wh.code,
        active: wh.active,
        allow_negative_stock: wh.allow_negative_stock ?? false,
        priority: wh.priority,
        address_line1: wh.address_line1 ?? '',
        address_line2: wh.address_line2 ?? '',
        city: wh.city ?? '',
        state: wh.state ?? '',
        postal_code: wh.postal_code ?? '',
        country: wh.country ?? '',
      };
      stockItems = stockRes.data ?? [];
    } catch (e) {
      notifications.error($t('warehouses.loadOneFailed'));
    } finally {
      loading = false;
      stockLoading = false;
    }
  });

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    if (!form.name.trim() || !form.code.trim()) {
      notifications.error($t('warehouses.nameAndCodeRequired'));
      return;
    }
    submitting = true;
    try {
      await warehousesApi.update(id, form);
      notifications.success($t('warehouses.saved'));
    } catch (e) {
      notifications.error($t('common.saveFailed'));
    } finally {
      submitting = false;
    }
  }

  async function handleDelete() {
    try {
      await warehousesApi.delete(id);
      notifications.success($t('warehouses.deleted'));
      goto(`${base}/warehouses`);
    } catch (e) {
      notifications.error($t('common.deleteFailed'));
    }
  }

  function startEditStock() {
    editingStock = true;
    stockEdits = {};
    for (const s of stockItems) {
      stockEdits[s.id] = s.quantity;
    }
  }

  function cancelEditStock() {
    editingStock = false;
    stockEdits = {};
  }

  async function saveStock() {
    savingStock = true;
    try {
      const items = stockItems
        .filter(s => stockEdits[s.id] !== undefined && stockEdits[s.id] !== s.quantity)
        .map(s => ({
          product_id: s.product_id,
          variant_id: s.variant_id ?? undefined,
          quantity: stockEdits[s.id],
          reference: 'admin-edit',
        }));
      if (items.length > 0) {
        await warehousesApi.setStock(id, items);
      }
      const stockRes = await warehousesApi.getStock(id);
      stockItems = stockRes.data ?? [];
      editingStock = false;
      notifications.success($t('warehouses.stockSaved'));
    } catch (e) {
      notifications.error($t('common.saveFailed'));
    } finally {
      savingStock = false;
    }
  }

  async function reloadStock() {
    const stockRes = await warehousesApi.getStock(id);
    stockItems = stockRes.data ?? [];
  }

  async function removeStockEntry(stock: any) {
    try {
      await warehousesApi.removeStock(id, stock.id);
      await reloadStock();
      notifications.success($t('warehouses.stockRemoved'));
    } catch {
      notifications.error($t('common.deleteFailed'));
    }
  }

  // Product + variant search with debounce
  function handleProductSearch() {
    clearTimeout(searchTimeout);
    if (productSearch.trim().length < 2) {
      searchResults = [];
      return;
    }
    searchLoading = true;
    searchTimeout = setTimeout(async () => {
      try {
        const res = await productsApi.list({ search: productSearch.trim(), limit: 10 });
        const products: any[] = (res as any).data?.items ?? (res as any).data ?? [];
        const results: SearchResult[] = [];
        for (const p of products) {
          const productName = tr(p.translations, 'name', $locale) || p.sku || p.id.slice(0, 8);
          // Add product-level entry
          results.push({
            product_id: p.id,
            label: productName,
            sku: p.sku || '—',
            isVariant: false,
          });
          // Add each variant as separate entry
          if (p.variants && p.variants.length > 0) {
            for (const v of p.variants) {
              results.push({
                product_id: p.id,
                variant_id: v.id,
                label: `${productName} — ${v.sku || v.id.slice(0, 8)}`,
                sku: v.sku || '—',
                isVariant: true,
              });
            }
          }
        }
        searchResults = results;
      } catch {
        searchResults = [];
      } finally {
        searchLoading = false;
      }
    }, 300);
  }

  function selectResult(result: SearchResult) {
    selectedResult = result;
    productSearch = '';
    searchResults = [];
    addQuantity = 0;
  }

  function openAddProduct() {
    showAddProduct = true;
    selectedResult = null;
    productSearch = '';
    searchResults = [];
    addQuantity = 0;
  }

  async function handleAddProduct(e: SubmitEvent) {
    e.preventDefault();
    if (!selectedResult || addQuantity < 0) return;
    addingProduct = true;
    try {
      const item: any = {
        product_id: selectedResult.product_id,
        quantity: addQuantity,
        reference: 'admin-add',
      };
      if (selectedResult.variant_id) {
        item.variant_id = selectedResult.variant_id;
      }
      await warehousesApi.setStock(id, [item]);
      await reloadStock();
      showAddProduct = false;
      notifications.success($t('warehouses.productAdded'));
    } catch (e) {
      notifications.error($t('common.saveFailed'));
    } finally {
      addingProduct = false;
    }
  }
</script>

<div class="mb-6">
  <a href="{base}/warehouses" class="text-sm text-primary-500 hover:text-primary-400 transition-colors">&larr; {$t('common.back')}</a>
</div>

{#if loading}
  <div class="flex items-center justify-center h-32">
    <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
  </div>
{:else}
  <div class="card p-6 max-w-2xl mb-6">
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-xl font-bold text-[var(--text)]">{$t('warehouses.editWarehouse')}</h1>
      <button class="btn btn-danger btn-sm" onclick={() => showDeleteConfirm = true}>{$t('common.delete')}</button>
    </div>

    <form onsubmit={handleSubmit} class="space-y-4">
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label class="label" for="name">{$t('common.name')}</label>
          <input id="name" class="input" type="text" bind:value={form.name} required />
        </div>
        <div>
          <label class="label" for="code">{$t('warehouses.code')}</label>
          <input id="code" class="input" type="text" bind:value={form.code} required />
        </div>
      </div>

      <div>
        <label class="label" for="priority">{$t('warehouses.priority')}</label>
        <input id="priority" class="input" type="number" min="0" bind:value={form.priority} />
        <p class="text-xs text-[var(--text-muted)] mt-1">{$t('warehouses.priorityHint')}</p>
      </div>

      <div class="border border-[var(--card-border)] rounded-lg p-4">
        <h3 class="text-sm font-semibold text-[var(--text-muted)] mb-3">{$t('warehouses.address')}</h3>
        <div class="space-y-3">
          <div>
            <label class="label" for="address_line1">{$t('warehouses.addressLine1')}</label>
            <input id="address_line1" class="input" type="text" bind:value={form.address_line1} />
          </div>
          <div>
            <label class="label" for="address_line2">{$t('warehouses.addressLine2')}</label>
            <input id="address_line2" class="input" type="text" bind:value={form.address_line2} />
          </div>
          <div class="grid grid-cols-3 gap-3">
            <div>
              <label class="label" for="postal_code">{$t('warehouses.postalCode')}</label>
              <input id="postal_code" class="input" type="text" bind:value={form.postal_code} />
            </div>
            <div>
              <label class="label" for="city">{$t('warehouses.city')}</label>
              <input id="city" class="input" type="text" bind:value={form.city} />
            </div>
            <div>
              <label class="label" for="country">{$t('warehouses.country')}</label>
              <input id="country" class="input" type="text" bind:value={form.country} maxlength="2" />
            </div>
          </div>
        </div>
      </div>

      <div class="flex flex-col gap-2">
        <div class="flex items-center gap-2">
          <input id="active" type="checkbox" bind:checked={form.active} class="h-4 w-4 rounded border-gray-300 text-primary-600" />
          <label for="active" class="text-sm text-[var(--text-muted)]">{$t('common.active')}</label>
        </div>
        <div class="flex items-center gap-2">
          <input id="allow_negative_stock" type="checkbox" bind:checked={form.allow_negative_stock} class="h-4 w-4 rounded border-gray-300 text-primary-600" />
          <div>
            <label for="allow_negative_stock" class="text-sm text-[var(--text-muted)]">{$t('warehouses.allow_negative_stock')}</label>
            <p class="text-xs text-[var(--text-muted)]">{$t('warehouses.allow_negative_stock_hint')}</p>
          </div>
        </div>
      </div>

      <div class="flex gap-3 pt-2">
        <button type="submit" class="btn btn-primary" disabled={submitting}>
          {submitting ? $t('common.saving') : $t('common.save')}
        </button>
        <a href="{base}/warehouses" class="btn btn-secondary">{$t('common.cancel')}</a>
      </div>
    </form>
  </div>

  <!-- Stock Overview -->
  <div class="card p-6 max-w-2xl">
    <div class="flex items-center justify-between mb-4">
      <h2 class="text-lg font-bold text-[var(--text)]">{$t('warehouses.stockOverview')}</h2>
      <div class="flex gap-2">
        <button class="btn btn-secondary btn-sm" onclick={openAddProduct}>{$t('warehouses.addProduct')}</button>
        {#if !editingStock && stockItems.length > 0}
          <button class="btn btn-secondary btn-sm" onclick={startEditStock}>{$t('common.edit')}</button>
        {/if}
      </div>
    </div>

    {#if stockLoading}
      <p class="text-[var(--text-muted)]">{$t('common.loading')}</p>
    {:else if stockItems.length === 0}
      <p class="text-[var(--text-muted)]">{$t('warehouses.noStock')}</p>
    {:else}
      <div class="overflow-x-auto">
        <table class="min-w-full divide-y divide-[var(--card-border)]">
          <thead>
            <tr>
              <th class="table-header">{$t('warehouses.sku')}</th>
              <th class="table-header">{$t('warehouses.productName')}</th>
              <th class="table-header">{$t('warehouses.variantSku')}</th>
              <th class="table-header">{$t('warehouses.quantity')}</th>
              <th class="px-4 py-2"></th>
            </tr>
          </thead>
          <tbody class="divide-y divide-[var(--card-border)]">
            {#each stockItems as stock}
              <tr class="table-row">
                <td class="table-cell font-mono text-sm text-[var(--text)]">{stock.product_sku || stock.product_id.slice(0, 8)}</td>
                <td class="table-cell text-sm text-[var(--text)]">{stock.product_name || '—'}</td>
                <td class="table-cell font-mono text-sm text-[var(--text-muted)]">{stock.variant_sku || '—'}</td>
                <td class="table-cell tabular-nums">
                  {#if editingStock}
                    <input type="number" min="0" class="input w-24" bind:value={stockEdits[stock.id]} />
                  {:else}
                    {stock.quantity}
                  {/if}
                </td>
                <td class="px-4 py-2 text-right">
                  {#if !editingStock}
                    <button class="text-red-600 hover:underline text-xs" onclick={() => removeStockEntry(stock)}>
                      {$t('common.delete')}
                    </button>
                  {/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>

      {#if editingStock}
        <div class="flex gap-3 mt-4">
          <button class="btn btn-primary btn-sm" onclick={saveStock} disabled={savingStock}>
            {savingStock ? $t('common.saving') : $t('common.save')}
          </button>
          <button class="btn btn-secondary btn-sm" onclick={cancelEditStock}>{$t('common.cancel')}</button>
        </div>
      {/if}
    {/if}
  </div>
{/if}

<ConfirmModal
  open={showDeleteConfirm}
  title={$t('warehouses.deleteTitle')}
  message={$t('warehouses.deleteMessage')}
  onConfirm={handleDelete}
  onCancel={() => showDeleteConfirm = false}
/>

<!-- Add Product / Variant Modal -->
<Modal open={showAddProduct} title={$t('warehouses.addProduct')} onClose={() => showAddProduct = false}>
  <form onsubmit={handleAddProduct} class="space-y-4">
    {#if !selectedResult}
      <div class="relative">
        <label class="label" for="product-search">{$t('warehouses.selectProduct')}</label>
        <input
          id="product-search"
          class="input"
          type="text"
          bind:value={productSearch}
          oninput={handleProductSearch}
          placeholder={$t('warehouses.selectProduct')}
          autocomplete="off"
        />
        {#if searchLoading}
          <div class="absolute right-3 top-9">
            <div class="animate-spin rounded-full h-4 w-4 border-b-2 border-primary-600"></div>
          </div>
        {/if}
        {#if searchResults.length > 0}
          <div class="absolute z-10 w-full mt-1 bg-[var(--surface)] border border-[var(--card-border)] rounded-lg shadow-lg max-h-60 overflow-y-auto">
            {#each searchResults as result}
              <button
                type="button"
                class="w-full text-left px-3 py-2 hover:bg-[var(--card-border)] transition-colors text-sm flex items-center gap-2"
                onclick={() => selectResult(result)}
              >
                {#if result.isVariant}
                  <span class="text-xs text-primary-500 font-medium shrink-0">VAR</span>
                {/if}
                <span class="font-mono text-[var(--text-muted)] shrink-0">{result.sku}</span>
                <span class="text-[var(--text)] truncate">{result.label}</span>
              </button>
            {/each}
          </div>
        {/if}
      </div>
    {:else}
      <div class="border border-[var(--card-border)] rounded-lg p-3 flex items-center justify-between">
        <div class="flex items-center gap-2 min-w-0">
          {#if selectedResult.isVariant}
            <span class="text-xs text-primary-500 font-medium shrink-0">VAR</span>
          {/if}
          <span class="font-mono text-sm text-[var(--text-muted)] shrink-0">{selectedResult.sku}</span>
          <span class="text-sm text-[var(--text)] truncate">{selectedResult.label}</span>
        </div>
        <button type="button" class="text-[var(--text-muted)] hover:text-[var(--text)] text-lg leading-none ml-2 shrink-0" onclick={() => selectedResult = null}>
          &times;
        </button>
      </div>

      <div>
        <label class="label" for="add-quantity">{$t('warehouses.quantity')}</label>
        <input id="add-quantity" class="input" type="number" min="0" bind:value={addQuantity} />
      </div>

      <div class="flex gap-3 pt-2">
        <button type="submit" class="btn btn-primary" disabled={addingProduct}>
          {addingProduct ? $t('common.saving') : $t('common.add')}
        </button>
        <button type="button" class="btn btn-secondary" onclick={() => showAddProduct = false}>{$t('common.cancel')}</button>
      </div>
    {/if}
  </form>
</Modal>
