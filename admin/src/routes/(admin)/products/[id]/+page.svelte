<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { page } from '$app/stores';
  import { productsApi } from '$lib/api/products';
  import { warehousesApi } from '$lib/api/warehouses';
  import { categoriesApi } from '$lib/api/categories';
  import { tagsApi } from '$lib/api/tags';
  import { mediaApi } from '$lib/api/media';
  import { taxApi } from '$lib/api/tax';
  import { propertyGroupsApi, type PropertyGroup } from '$lib/api/property-groups';
  import { notifications } from '$lib/stores/notifications';
  import type { Media } from '$lib/types';
  import Modal from '$lib/components/Modal.svelte';
  import ConfirmModal from '$lib/components/ConfirmModal.svelte';
  import TranslationsInput from '$lib/components/TranslationsInput.svelte';
  import {
    AVAILABLE_LOCALES,
    DEFAULT_LOCALE,
    LOCALE_LABELS,
    emptyTranslations,
    translationsFromArray,
    translationsToArray,
  } from '$lib/config';
  import { t, locale } from 'svelte-i18n';
  import { fmt } from '$lib/i18n/formatters';
  import { tr } from '$lib/i18n/entity';

  const FIELDS = ['name', 'slug', 'description', 'meta_title', 'meta_description'];

  let id = $derived($page.params.id as string);
  let loading = $state(true);
  let submitting = $state(false);
  let showDeleteConfirm = $state(false);
  let showVariantModal = $state(false);
  let showGenerateModal = $state(false);

  let form = $state({
    sku: '',
    active: true,
  });

  let translations = $state(emptyTranslations(FIELDS));
  let variants = $state<any[]>([]);
  let allCategories = $state<any[]>([]);
  let selectedCategoryIds = $state<string[]>([]);
  let allTags = $state<any[]>([]);
  let selectedTagIds = $state<string[]>([]);

  // Eigenschaftsgruppen
  let allPropertyGroups = $state<PropertyGroup[]>([]);
  // Für "Variante hinzufügen" Modal: groupId → optionId
  let addVariantOptions = $state<Record<string, string>>({});

  let variantForm = $state({
    sku: '',
    price_gross: 0,
    stock: 0,
    active: true,
  });
  let variantSubmitting = $state(false);

  // Variante bearbeiten
  let editVariant = $state<any | null>(null);
  let editVariantOptions = $state<Record<string, string>>({});
  let editVariantForm = $state({ sku: '', price_gross: 0, stock: 0, active: true });
  let editVariantSubmitting = $state(false);

  // Variante löschen
  let deleteVariantId = $state<string | null>(null);

  // "Alle Kombinationen" generieren
  let generateSelections = $state<Record<string, string[]>>({});
  let generateSubmitting = $state(false);

  let allMedia = $state<Media[]>([]);
  let productMediaIds = $state<string[]>([]);
  let showMediaPicker = $state(false);

  // Warehouse stock
  let warehouseStock = $state<any[]>([]);
  let editingWarehouseStock = $state(false);
  let warehouseStockEdits = $state<Record<string, number>>({});
  let savingWarehouseStock = $state(false);

  let allTaxRules = $state<any[]>([]);
  let selectedTaxRuleId = $state('');
  let priceMode = $state<'gross' | 'net'>('gross');
  let enteredPrice = $state(0);

  const selectedTaxRule = $derived(allTaxRules.find(t => t.id === selectedTaxRuleId));
  const calculatedPrice = $derived((() => {
    if (!selectedTaxRule || !enteredPrice) return null;
    const rate = selectedTaxRule.rate;
    return priceMode === 'gross'
      ? Math.round(enteredPrice * 10000 / (10000 + rate))
      : Math.round(enteredPrice * (10000 + rate) / 10000);
  })());

  onMount(async () => {
    try {
      const [res, catRes, tagRes, mediaRes, taxRes, pgRes] = await Promise.all([
        productsApi.get(id),
        categoriesApi.list({ limit: 200 }),
        tagsApi.list({ limit: 200 }),
        mediaApi.list({ limit: 200 }),
        taxApi.list({ limit: 200 }),
        propertyGroupsApi.list(),
      ]);
      const product = res.data;
      form = {
        sku: product.sku ?? '',
        active: product.active ?? true,
      };
      translations = translationsFromArray(product.translations, FIELDS);
      variants = product.variants ?? [];
      allCategories = catRes.data ?? [];
      selectedCategoryIds = (product.categories ?? []) as string[];
      allTags = (tagRes as any).data ?? [];
      selectedTagIds = (product.tags ?? []) as string[];
      allMedia = (mediaRes as any).data ?? [];
      productMediaIds = (product.media ?? []).map((m: any) => m.media_id);
      allTaxRules = (taxRes as any).data ?? [];
      selectedTaxRuleId = product.tax_rule_id ?? '';
      allPropertyGroups = pgRes.data ?? [];
      // Load warehouse stock (non-fatal).
      try {
        const whRes = await warehousesApi.getProductStock(id);
        warehouseStock = whRes.data ?? [];
      } catch { /* non-fatal */ }
      if (product.price_net > 0 && product.price_gross === 0) {
        priceMode = 'net';
        enteredPrice = product.price_net;
      } else {
        priceMode = 'gross';
        enteredPrice = product.price_gross ?? 0;
      }
    } catch {
      notifications.error($t('products.loadOneFailed'));
    } finally {
      loading = false;
    }
  });

  function toggleTag(tagId: string) {
    if (selectedTagIds.includes(tagId)) {
      selectedTagIds = selectedTagIds.filter((t) => t !== tagId);
    } else {
      selectedTagIds = [...selectedTagIds, tagId];
    }
  }

  function toggleCategory(catId: string) {
    if (selectedCategoryIds.includes(catId)) {
      selectedCategoryIds = selectedCategoryIds.filter((c) => c !== catId);
    } else {
      selectedCategoryIds = [...selectedCategoryIds, catId];
    }
  }

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    if (!translations[DEFAULT_LOCALE].name.trim()) {
      notifications.error($t('common.pleaseNameInLocale', { values: { locale: LOCALE_LABELS[DEFAULT_LOCALE] } }));
      return;
    }
    submitting = true;
    try {
      await productsApi.update(id, {
        sku: form.sku,
        active: form.active,
        price_gross: priceMode === 'gross' ? Number(enteredPrice) : 0,
        price_net: priceMode === 'net' ? Number(enteredPrice) : 0,
        tax_rule_id: selectedTaxRuleId || null,
        category_ids: selectedCategoryIds,
        tag_ids: selectedTagIds,
        translations: translationsToArray(translations),
        media_ids: productMediaIds,
      } as any);
      notifications.success($t('products.saved'));
    } catch {
      notifications.error($t('common.saveFailed'));
    } finally {
      submitting = false;
    }
  }

  async function handleDelete() {
    try {
      await productsApi.delete(id);
      notifications.success($t('products.deleted'));
      goto(`${base}/products`);
    } catch {
      notifications.error($t('common.deleteFailed'));
    }
  }

  function toggleMedia(mediaId: string) {
    if (productMediaIds.includes(mediaId)) {
      productMediaIds = productMediaIds.filter((m) => m !== mediaId);
    } else {
      productMediaIds = [...productMediaIds, mediaId];
    }
  }

  function mediaById(mediaId: string): Media | undefined {
    return allMedia.find((m) => m.id === mediaId);
  }

  // Hilfsfunktion: Optionsname für eine Ausprägung
  function optionName(groupId: string, optionId: string): string {
    const group = allPropertyGroups.find((g) => g.id === groupId);
    const opt = group?.options?.find((o) => o.id === optionId);
    if (!opt) return optionId;
    return tr(opt.translations, 'name', $locale) || opt.id;
  }

  // Variant-Optionen als lesbaren String
  function variantOptionsLabel(v: any): string {
    if (!v.options || v.options.length === 0) return '—';
    return v.options
      .map((o: any) => {
        const group = allPropertyGroups.find((g) => g.id === o.group_id);
        const opt = group?.options?.find((op) => op.id === o.id);
        const name = tr(opt?.translations, 'name', $locale) || o.id;
        return name;
      })
      .join(', ');
  }

  async function handleVariantSubmit(e: SubmitEvent) {
    e.preventDefault();
    variantSubmitting = true;
    try {
      await productsApi.createVariant(id, {
        ...variantForm,
        price_gross: Number(variantForm.price_gross),
        stock: Number(variantForm.stock),
        option_ids: Object.values(addVariantOptions).filter(Boolean),
      } as any);
      notifications.success($t('products.variantAdded'));
      showVariantModal = false;
      variantForm = { sku: '', price_gross: 0, stock: 0, active: true };
      addVariantOptions = {};
      const product = await productsApi.get(id);
      variants = product.data.variants ?? [];
    } catch {
      notifications.error($t('products.variantCreateFailed'));
    } finally {
      variantSubmitting = false;
    }
  }

  function openEditVariant(v: any) {
    editVariant = v;
    editVariantForm = {
      sku: v.sku ?? '',
      price_gross: v.price_gross ?? 0,
      stock: v.stock ?? 0,
      active: v.active ?? true,
    };
    // Vorauswahl der Optionen: groupId → optionId
    editVariantOptions = {};
    for (const o of v.options ?? []) {
      editVariantOptions[o.group_id] = o.id;
    }
  }

  async function handleEditVariantSubmit(e: SubmitEvent) {
    e.preventDefault();
    if (!editVariant) return;
    editVariantSubmitting = true;
    try {
      await productsApi.updateVariant(id, editVariant.id, {
        ...editVariantForm,
        price_gross: Number(editVariantForm.price_gross),
        stock: Number(editVariantForm.stock),
        option_ids: Object.values(editVariantOptions).filter(Boolean),
      } as any);
      notifications.success($t('products.variantSaved'));
      editVariant = null;
      const product = await productsApi.get(id);
      variants = product.data.variants ?? [];
    } catch {
      notifications.error($t('common.saveFailed'));
    } finally {
      editVariantSubmitting = false;
    }
  }

  async function handleDeleteVariant() {
    if (!deleteVariantId) return;
    try {
      await productsApi.deleteVariant(id, deleteVariantId);
      variants = variants.filter((v) => v.id !== deleteVariantId);
      deleteVariantId = null;
      notifications.success($t('products.variantDeleted'));
    } catch {
      notifications.error($t('common.deleteFailed'));
    }
  }

  async function handleGenerateVariants(e: SubmitEvent) {
    e.preventDefault();
    const optionGroups = allPropertyGroups
      .map((g) => (generateSelections[g.id] ?? []))
      .filter((arr) => arr.length > 0);

    if (optionGroups.length === 0) {
      notifications.error($t('products.selectAtLeastOneOption'));
      return;
    }
    generateSubmitting = true;
    try {
      await productsApi.createVariant(id, { option_groups: optionGroups } as any);
      notifications.success($t('products.variantsGenerated'));
      showGenerateModal = false;
      generateSelections = {};
      const product = await productsApi.get(id);
      variants = product.data.variants ?? [];
    } catch {
      notifications.error($t('products.variantsGenerateFailed'));
    } finally {
      generateSubmitting = false;
    }
  }

  function toggleGenerateOption(groupId: string, optId: string) {
    const current = generateSelections[groupId] ?? [];
    if (current.includes(optId)) {
      generateSelections = { ...generateSelections, [groupId]: current.filter((o) => o !== optId) };
    } else {
      generateSelections = { ...generateSelections, [groupId]: [...current, optId] };
    }
  }

  function startEditWarehouseStock() {
    editingWarehouseStock = true;
    warehouseStockEdits = {};
    for (const s of warehouseStock) {
      warehouseStockEdits[s.id] = s.quantity;
    }
  }

  function cancelEditWarehouseStock() {
    editingWarehouseStock = false;
    warehouseStockEdits = {};
  }

  async function saveWarehouseStock() {
    savingWarehouseStock = true;
    try {
      // Group changes by warehouse_id
      const byWarehouse: Record<string, any[]> = {};
      for (const s of warehouseStock) {
        if (warehouseStockEdits[s.id] !== undefined && warehouseStockEdits[s.id] !== s.quantity) {
          if (!byWarehouse[s.warehouse_id]) byWarehouse[s.warehouse_id] = [];
          byWarehouse[s.warehouse_id].push({
            product_id: s.product_id,
            variant_id: s.variant_id ?? undefined,
            quantity: warehouseStockEdits[s.id],
            reference: 'admin-product-edit',
          });
        }
      }
      for (const [whId, items] of Object.entries(byWarehouse)) {
        await warehousesApi.setStock(whId, items);
      }
      const whRes = await warehousesApi.getProductStock(id);
      warehouseStock = whRes.data ?? [];
      editingWarehouseStock = false;
      notifications.success($t('warehouses.stockSaved'));
    } catch {
      notifications.error($t('common.saveFailed'));
    } finally {
      savingWarehouseStock = false;
    }
  }
</script>

<div class="mb-6">
  <a href="{base}/products" class="text-sm text-primary-500 hover:text-primary-400 transition-colors">&larr; {$t('common.back')}</a>
</div>

{#if loading}
  <div class="flex items-center justify-center h-32">
    <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
  </div>
{:else}
  <div class="card p-6 max-w-2xl mb-6">
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-xl font-bold text-[var(--text)]">{$t('products.editProduct')}</h1>
      <button class="btn btn-danger btn-sm" onclick={() => showDeleteConfirm = true}>{$t('common.delete')}</button>
    </div>

    <form onsubmit={handleSubmit} class="space-y-4">
      <div>
        <label class="label" for="sku">{$t('products.sku')} *</label>
        <input id="sku" class="input" type="text" bind:value={form.sku} required />
      </div>

      <div class="border border-[var(--card-border)] rounded-lg p-4">
        <h3 class="text-sm font-semibold text-[var(--text-muted)] mb-3">{$t('common.translations')}</h3>
        <TranslationsInput
          locales={AVAILABLE_LOCALES}
          localeLabels={LOCALE_LABELS}
          primaryLocale={DEFAULT_LOCALE}
          fields={[
            { key: 'name', label: $t('common.name'), type: 'input', required: true },
            { key: 'slug', label: $t('common.slug'), type: 'input', required: true },
            { key: 'description', label: $t('common.description'), type: 'textarea', rows: 4 },
            { key: 'meta_title', label: $t('products.metaTitle'), type: 'input' },
            { key: 'meta_description', label: $t('products.metaDescription'), type: 'textarea', rows: 2 },
          ]}
          bind:value={translations}
        />
      </div>

      <!-- Steuerregel -->
      <div>
        <label class="label" for="tax_rule">{$t('products.taxRule')}</label>
        <select id="tax_rule" class="input" bind:value={selectedTaxRuleId}>
          <option value="">{$t('products.noTaxRule')}</option>
          {#each allTaxRules as t}
            <option value={t.id}>{t.name} ({t.rate / 100}%)</option>
          {/each}
        </select>
      </div>

      {#if selectedTaxRule}
        <div class="flex gap-4">
          <label class="flex items-center gap-2 cursor-pointer text-sm text-[var(--text-muted)]">
            <input type="radio" bind:group={priceMode} value="gross" />
            {$t('products.grossInput')}
          </label>
          <label class="flex items-center gap-2 cursor-pointer text-sm text-[var(--text-muted)]">
            <input type="radio" bind:group={priceMode} value="net" />
            {$t('products.netInput')}
          </label>
        </div>
      {/if}

      <div>
        <label class="label" for="entered_price">
          {selectedTaxRule ? (priceMode === 'gross' ? $t('products.grossLabel') : $t('products.netLabel')) : $t('products.priceGrossLabel')} {$t('products.priceCents')}
        </label>
        <input id="entered_price" class="input" type="number" min="0" bind:value={enteredPrice}
          placeholder={$t('products.grossPlaceholder')} />
      </div>

      {#if calculatedPrice !== null}
        <p class="text-sm text-[var(--text-muted)]">
          {priceMode === 'gross' ? $t('products.netCalculated') : $t('products.grossCalculated')}:
          {$fmt.price(calculatedPrice)}
        </p>
      {/if}

      <div>
        <p class="label">{$t('products.categories')}</p>
        {#if allCategories.length === 0}
          <p class="text-sm text-[var(--text-muted)]">{$t('products.noCategories')}</p>
        {:else}
          <div class="border border-[var(--card-border)] rounded-md p-3 space-y-1 max-h-48 overflow-y-auto">
            {#each allCategories as cat}
              <label class="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  class="h-4 w-4 rounded border-gray-300 text-primary-600"
                  checked={selectedCategoryIds.includes(cat.id)}
                  onchange={() => toggleCategory(cat.id)}
                />
                <span class="text-sm text-[var(--text)]">{tr(cat.translations, 'name', $locale) || cat.id}</span>
              </label>
            {/each}
          </div>
        {/if}
      </div>

      <div>
        <p class="label">{$t('products.tags')}</p>
        {#if allTags.length === 0}
          <p class="text-sm text-[var(--text-muted)]">{$t('products.noTags')}</p>
        {:else}
          <div class="border border-[var(--card-border)] rounded-md p-3 space-y-1 max-h-48 overflow-y-auto">
            {#each allTags as tag}
              <label class="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  class="h-4 w-4 rounded border-gray-300 text-primary-600"
                  checked={selectedTagIds.includes(tag.id)}
                  onchange={() => toggleTag(tag.id)}
                />
                <span class="text-sm text-[var(--text)]">{tag.name}</span>
              </label>
            {/each}
          </div>
        {/if}
      </div>

      <div class="flex items-center gap-2">
        <input id="active" type="checkbox" bind:checked={form.active} class="h-4 w-4 rounded border-gray-300 text-primary-600" />
        <label for="active" class="text-sm text-[var(--text-muted)]">{$t('common.active')}</label>
      </div>

      <div class="flex gap-3 pt-2">
        <button type="submit" class="btn btn-primary" disabled={submitting}>
          {submitting ? $t('common.saving') : $t('common.save')}
        </button>
        <a href="{base}/products" class="btn btn-secondary">{$t('common.cancel')}</a>
      </div>
    </form>
  </div>

  <!-- Media -->
  <div class="card p-6 max-w-2xl mb-6">
    <div class="flex items-center justify-between mb-4">
      <h2 class="text-lg font-semibold text-[var(--text)]">{$t('products.images')}</h2>
      <button class="btn btn-secondary btn-sm" onclick={() => showMediaPicker = !showMediaPicker}>
        {showMediaPicker ? $t('products.closeSelection') : $t('products.addImage')}
      </button>
    </div>

    {#if productMediaIds.length === 0 && !showMediaPicker}
      <p class="text-sm text-[var(--text-muted)]">{$t('products.noImages')}</p>
    {/if}

    {#if productMediaIds.length > 0}
      <div class="grid grid-cols-3 sm:grid-cols-4 gap-3 mb-4">
        {#each productMediaIds as mid}
          {@const item = mediaById(mid)}
          <div class="relative group">
            <div class="aspect-square rounded-lg overflow-hidden bg-gray-100 dark:bg-gray-800 flex items-center justify-center">
              {#if item?.mime_type?.startsWith('image/') && item.url}
                <img src={item.url} alt={item.filename} class="w-full h-full object-cover" />
              {:else}
                <svg class="h-8 w-8 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
              {/if}
            </div>
            <button
              class="absolute top-1 right-1 opacity-0 group-hover:opacity-100 transition-opacity bg-red-500 text-white rounded-full w-5 h-5 flex items-center justify-center text-xs"
              onclick={() => toggleMedia(mid)}
              title={$t('products.remove')}
            >&times;</button>
            <p class="text-xs text-[var(--text-muted)] truncate mt-1">{item?.filename ?? mid}</p>
          </div>
        {/each}
      </div>
    {/if}

    {#if showMediaPicker}
      {#if allMedia.length === 0}
        <p class="text-sm text-[var(--text-muted)]">{$t('products.noMediaAvailable', { values: { link: '' } })}<a href="{base}/media" class="text-primary-500 underline">{$t('products.mediaLinkLabel')}</a></p>
      {:else}
        <div class="grid grid-cols-3 sm:grid-cols-4 gap-3 max-h-64 overflow-y-auto border border-[var(--card-border)] rounded-lg p-3">
          {#each allMedia as item}
            <button
              onclick={() => toggleMedia(item.id)}
              class="relative rounded-lg overflow-hidden border-2 transition-colors {productMediaIds.includes(item.id) ? 'border-primary-500' : 'border-[var(--card-border)] hover:border-gray-400'}"
              title={item.filename}
            >
              <div class="aspect-square bg-gray-100 dark:bg-gray-800 flex items-center justify-center">
                {#if item.mime_type?.startsWith('image/') && item.url}
                  <img src={item.url} alt={item.filename} class="w-full h-full object-cover" />
                {:else}
                  <svg class="h-8 w-8 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                  </svg>
                {/if}
              </div>
              {#if productMediaIds.includes(item.id)}
                <div class="absolute inset-0 bg-primary-500/20 flex items-center justify-center">
                  <svg class="h-6 w-6 text-primary-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
                  </svg>
                </div>
              {/if}
            </button>
          {/each}
        </div>
        <p class="text-xs text-[var(--text-muted)] mt-2">{$t('common.clickToSelectDeselect')}</p>
      {/if}
    {/if}
  </div>

  <!-- Variants -->
  <div class="card p-6 max-w-2xl">
    <div class="flex items-center justify-between mb-4">
      <h2 class="text-lg font-semibold text-[var(--text)]">{$t('products.variants')}</h2>
      <div class="flex gap-2">
        {#if allPropertyGroups.length > 0}
          <button class="btn btn-secondary btn-sm" onclick={() => showGenerateModal = true}>
            {$t('products.allCombinations')}
          </button>
        {/if}
        <button class="btn btn-secondary btn-sm" onclick={() => showVariantModal = true}>
          {$t('products.addVariant')}
        </button>
      </div>
    </div>

    {#if variants.length === 0}
      <p class="text-sm text-[var(--text-muted)]">{$t('products.noVariants')}</p>
    {:else}
      <div class="overflow-x-auto">
        <table class="min-w-full divide-y divide-[var(--card-border)]">
          <thead>
            <tr>
              <th class="table-header">{$t('products.sku')}</th>
              <th class="table-header">{$t('products.properties')}</th>
              <th class="table-header">{$t('common.price')}</th>
              <th class="table-header">{$t('products.stock')}</th>
              <th class="table-header">{$t('common.active')}</th>
              <th class="px-4 py-2"></th>
            </tr>
          </thead>
          <tbody class="divide-y divide-[var(--card-border)]">
            {#each variants as v}
              <tr>
                <td class="table-cell text-[var(--text)]">{v.sku || '—'}</td>
                <td class="table-cell text-[var(--text-muted)]">{variantOptionsLabel(v)}</td>
                <td class="table-cell text-[var(--text)]">{$fmt.price(v.price_gross)}</td>
                <td class="table-cell text-[var(--text)]">{v.stock ?? 0}</td>
                <td class="px-4 py-2 text-sm">
                  {#if v.active}
                    <span class="badge badge-green">{$t('common.active')}</span>
                  {:else}
                    <span class="badge badge-gray">{$t('common.inactive')}</span>
                  {/if}
                </td>
                <td class="px-4 py-2 text-right flex gap-2 justify-end">
                  <button class="text-primary-500 hover:text-primary-400 transition-colors text-xs" onclick={() => openEditVariant(v)}>
                    {$t('common.edit')}
                  </button>
                  <button class="text-red-600 hover:underline text-xs" onclick={() => deleteVariantId = v.id}>
                    {$t('common.delete')}
                  </button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
{/if}

<ConfirmModal
  open={showDeleteConfirm}
  title={$t('products.deleteTitle')}
  message={$t('products.deleteMessage')}
  onConfirm={handleDelete}
  onCancel={() => showDeleteConfirm = false}
/>

<ConfirmModal
  open={!!deleteVariantId}
  title={$t('products.variantDeleteTitle')}
  message={$t('products.variantDeleteMessage')}
  onConfirm={handleDeleteVariant}
  onCancel={() => deleteVariantId = null}
/>

<!-- Variante hinzufügen -->
<Modal open={showVariantModal} title={$t('products.addVariant')} onClose={() => showVariantModal = false}>
  <form onsubmit={handleVariantSubmit} class="space-y-4">
    {#if allPropertyGroups.length > 0}
      <div>
        <p class="text-sm font-medium text-[var(--text-muted)] mb-2">{$t('products.properties')}</p>
        {#each allPropertyGroups as g}
          {@const gName = tr(g.translations, 'name', $locale) || g.id}
          <div class="mb-2">
            <label class="text-xs text-[var(--text-muted)] uppercase" for="add-variant-{g.id}">{gName}</label>
            <select id="add-variant-{g.id}" class="input mt-1" bind:value={addVariantOptions[g.id]}>
              <option value="">{$t('products.noSelection')}</option>
              {#each g.options ?? [] as o}
                {@const oName = tr(o.translations, 'name', $locale) || o.id}
                <option value={o.id}>{oName}</option>
              {/each}
            </select>
          </div>
        {/each}
      </div>
    {/if}
    <div>
      <label class="label" for="v-sku">{$t('products.variantSku')}</label>
      <input id="v-sku" class="input" type="text" bind:value={variantForm.sku} />
    </div>
    <div>
      <label class="label" for="v-price">{$t('products.variantPriceGross')}</label>
      <input id="v-price" class="input" type="number" min="0" bind:value={variantForm.price_gross} placeholder="1999 = 19,99 €" />
    </div>
    <div>
      <label class="label" for="v-stock">{$t('products.variantStock')}</label>
      <input id="v-stock" class="input" type="number" min="0" bind:value={variantForm.stock} />
    </div>
    <div class="flex items-center gap-2">
      <input id="v-active" type="checkbox" bind:checked={variantForm.active} class="h-4 w-4 rounded border-gray-300 text-primary-600" />
      <label for="v-active" class="text-sm text-[var(--text-muted)]">{$t('common.active')}</label>
    </div>
    <div class="flex gap-3 pt-2">
      <button type="submit" class="btn btn-primary" disabled={variantSubmitting}>
        {variantSubmitting ? $t('common.saving') : $t('common.add')}
      </button>
      <button type="button" class="btn btn-secondary" onclick={() => showVariantModal = false}>{$t('common.cancel')}</button>
    </div>
  </form>
</Modal>

<!-- Variante bearbeiten -->
{#if editVariant}
  <Modal open={true} title={$t('common.edit')} onClose={() => editVariant = null}>
    <form onsubmit={handleEditVariantSubmit} class="space-y-4">
      {#if allPropertyGroups.length > 0}
        <div>
          <p class="text-sm font-medium text-[var(--text-muted)] mb-2">{$t('products.properties')}</p>
          {#each allPropertyGroups as g}
            {@const gName = tr(g.translations, 'name', $locale) || g.id}
            <div class="mb-2">
              <label class="text-xs text-[var(--text-muted)] uppercase" for="edit-variant-{g.id}">{gName}</label>
              <select id="edit-variant-{g.id}" class="input mt-1" bind:value={editVariantOptions[g.id]}>
                <option value="">{$t('products.noSelection')}</option>
                {#each g.options ?? [] as o}
                  {@const oName = tr(o.translations, 'name', $locale) || o.id}
                  <option value={o.id}>{oName}</option>
                {/each}
              </select>
            </div>
          {/each}
        </div>
      {/if}
      <div>
        <label class="label" for="edit-v-sku">{$t('products.variantSku')}</label>
        <input id="edit-v-sku" class="input" type="text" bind:value={editVariantForm.sku} />
      </div>
      <div>
        <label class="label" for="edit-v-price">{$t('products.variantPriceGross')}</label>
        <input id="edit-v-price" class="input" type="number" min="0" bind:value={editVariantForm.price_gross} />
      </div>
      <div>
        <label class="label" for="edit-v-stock">{$t('products.variantStock')}</label>
        <input id="edit-v-stock" class="input" type="number" min="0" bind:value={editVariantForm.stock} />
      </div>
      <div class="flex items-center gap-2">
        <input id="edit-v-active" type="checkbox" bind:checked={editVariantForm.active} class="h-4 w-4 rounded border-gray-300 text-primary-600" />
        <label for="edit-v-active" class="text-sm text-[var(--text-muted)]">{$t('common.active')}</label>
      </div>
      <div class="flex gap-3 pt-2">
        <button type="submit" class="btn btn-primary" disabled={editVariantSubmitting}>
          {editVariantSubmitting ? $t('common.saving') : $t('common.save')}
        </button>
        <button type="button" class="btn btn-secondary" onclick={() => editVariant = null}>{$t('common.cancel')}</button>
      </div>
    </form>
  </Modal>
{/if}

<!-- Warehouse Stock -->
{#if warehouseStock.length > 0}
  <div class="card p-6 max-w-2xl mt-6">
    <div class="flex items-center justify-between mb-4">
      <h2 class="text-lg font-bold text-[var(--text)]">{$t('warehouses.stockOverview')}</h2>
      {#if !editingWarehouseStock}
        <button class="btn btn-secondary btn-sm" onclick={startEditWarehouseStock}>{$t('common.edit')}</button>
      {/if}
    </div>
    <div class="overflow-x-auto">
      <table class="min-w-full divide-y divide-[var(--card-border)]">
        <thead>
          <tr>
            <th class="table-header">{$t('warehouses.title')}</th>
            <th class="table-header">{$t('warehouses.code')}</th>
            <th class="table-header">{$t('warehouses.variantSku')}</th>
            <th class="table-header">{$t('warehouses.quantity')}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-[var(--card-border)]">
          {#each warehouseStock as stock}
            <tr class="table-row">
              <td class="table-cell text-[var(--text)]">{stock.warehouse_name || stock.warehouse_id.slice(0, 8)}</td>
              <td class="table-cell font-mono text-sm text-[var(--text-muted)]">{stock.warehouse_code || '—'}</td>
              <td class="table-cell font-mono text-sm text-[var(--text-muted)]">{stock.variant_sku || '—'}</td>
              <td class="table-cell tabular-nums">
                {#if editingWarehouseStock}
                  <input type="number" min="0" class="input w-24" bind:value={warehouseStockEdits[stock.id]} />
                {:else}
                  {stock.quantity}
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
    {#if editingWarehouseStock}
      <div class="flex gap-3 mt-4">
        <button class="btn btn-primary btn-sm" onclick={saveWarehouseStock} disabled={savingWarehouseStock}>
          {savingWarehouseStock ? $t('common.saving') : $t('common.save')}
        </button>
        <button class="btn btn-secondary btn-sm" onclick={cancelEditWarehouseStock}>{$t('common.cancel')}</button>
      </div>
    {/if}
  </div>
{/if}

<!-- Alle Kombinationen generieren -->
<Modal open={showGenerateModal} title={$t('products.generateCombinations')} onClose={() => showGenerateModal = false}>
  <form onsubmit={handleGenerateVariants} class="space-y-4">
    <p class="text-sm text-[var(--text-muted)]">{$t('products.generateDescription')}</p>
    {#each allPropertyGroups as g}
      {#if (g.options?.length ?? 0) > 0}
        {@const gName = tr(g.translations, 'name', $locale) || g.id}
        <div>
          <p class="text-sm font-medium text-[var(--text-muted)] mb-2">{gName}</p>
          <div class="flex flex-wrap gap-2">
            {#each g.options ?? [] as o}
              {@const oName = tr(o.translations, 'name', $locale) || o.id}
              <label class="flex items-center gap-2 cursor-pointer text-sm text-[var(--text-muted)] border border-[var(--card-border)] rounded px-2 py-1">
                <input
                  type="checkbox"
                  class="h-4 w-4 rounded border-gray-300 text-primary-600"
                  checked={(generateSelections[g.id] ?? []).includes(o.id)}
                  onchange={() => toggleGenerateOption(g.id, o.id)}
                />
                {#if o.color_hex}
                  <span class="w-3 h-3 rounded-full border border-gray-300 inline-block" style="background:{o.color_hex}"></span>
                {/if}
                {oName}
              </label>
            {/each}
          </div>
        </div>
      {/if}
    {/each}
    <div class="flex gap-3 pt-2">
      <button type="submit" class="btn btn-primary" disabled={generateSubmitting}>
        {generateSubmitting ? $t('products.generating') : $t('products.createCombinations')}
      </button>
      <button type="button" class="btn btn-secondary" onclick={() => showGenerateModal = false}>{$t('common.cancel')}</button>
    </div>
  </form>
</Modal>
