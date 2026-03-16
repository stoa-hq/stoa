<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { t, locale } from 'svelte-i18n';
  import { categoriesApi } from '$lib/api/categories';
  import { tr } from '$lib/i18n/entity';
  import { notifications } from '$lib/stores/notifications';
  import ConfirmModal from '$lib/components/ConfirmModal.svelte';
  import SearchBar from '$lib/components/SearchBar.svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';

  let items = $state<any[]>([]);
  let loading = $state(true);
  let deleteId = $state<string | null>(null);
  let showConfirm = $state(false);
  let search = $state('');

  const filtered = $derived(
    search
      ? items.filter(i => {
          const name = tr(i.translations, 'name', $locale) || '';
          return name.toLowerCase().includes(search.toLowerCase());
        })
      : items
  );

  async function load() {
    loading = true;
    try {
      const res = await categoriesApi.list({ limit: 100 });
      items = res.data ?? [];
    } catch (e) {
      notifications.error($t('categories.loadFailed'));
    } finally {
      loading = false;
    }
  }

  onMount(load);

  function getParentName(parentId: string | null) {
    if (!parentId) return '—';
    const parent = items.find(i => i.id === parentId);
    return tr(parent?.translations, 'name', $locale) || parentId;
  }

  function confirmDelete(id: string, e: MouseEvent) {
    e.stopPropagation();
    deleteId = id;
    showConfirm = true;
  }

  async function doDelete() {
    if (!deleteId) return;
    try {
      await categoriesApi.delete(deleteId);
      notifications.success($t('categories.deleted'));
      showConfirm = false;
      deleteId = null;
      load();
    } catch (e) {
      notifications.error($t('common.deleteFailed'));
    }
  }
</script>

<div class="flex items-center justify-between mb-6">
  <h1 class="text-2xl font-bold text-[var(--text)]">{$t('categories.title')}</h1>
  <a href="{base}/categories/new" class="btn btn-primary">{$t('common.new')}</a>
</div>

<div class="card p-6">
  <div class="mb-4">
    <SearchBar value={search} onSearch={(v) => search = v} debounce={200} />
  </div>

  {#if loading}
    <div class="space-y-3">
      {#each Array(5) as _}
        <Skeleton height="h-12" />
      {/each}
    </div>
  {:else}
    <div class="overflow-x-auto">
      <table class="min-w-full divide-y divide-[var(--card-border)]">
        <thead>
          <tr>
            <th class="table-header">{$t('common.name')}</th>
            <th class="table-header">{$t('common.slug')}</th>
            <th class="table-header">{$t('categories.parent')}</th>
            <th class="table-header">{$t('common.active')}</th>
            <th class="table-header">{$t('common.position')}</th>
            <th class="table-header"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-[var(--card-border)]">
          {#each filtered as item}
            <tr class="table-row cursor-pointer" onclick={() => goto(`${base}/categories/${item.id}`)}>
              <td class="table-cell font-medium text-[var(--text)]">{tr(item.translations, 'name', $locale) || item.id}</td>
              <td class="table-cell text-[var(--text-muted)]">{tr(item.translations, 'slug', $locale)}</td>
              <td class="table-cell text-[var(--text-muted)]">{getParentName(item.parent_id)}</td>
              <td class="table-cell">
                {#if item.active}
                  <span class="badge badge-green">{$t('common.active')}</span>
                {:else}
                  <span class="badge badge-gray">{$t('common.inactive')}</span>
                {/if}
              </td>
              <td class="table-cell text-[var(--text-muted)] tabular-nums">{item.position ?? 0}</td>
              <td class="table-cell text-right">
                <button class="btn btn-danger btn-sm" onclick={(e) => confirmDelete(item.id, e)}>{$t('common.delete')}</button>
              </td>
            </tr>
          {/each}
          {#if filtered.length === 0}
            <tr>
              <td colspan="6" class="table-cell text-center text-[var(--text-muted)] py-6">{$t('categories.noCategories')}</td>
            </tr>
          {/if}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<ConfirmModal
  open={showConfirm}
  title={$t('categories.deleteTitle')}
  message={$t('categories.deleteMessage')}
  onConfirm={doDelete}
  onCancel={() => { showConfirm = false; deleteId = null; }}
/>
