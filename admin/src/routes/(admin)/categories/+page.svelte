<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { categoriesApi } from '$lib/api/categories';
  import { notifications } from '$lib/stores/notifications';
  import ConfirmModal from '$lib/components/ConfirmModal.svelte';

  let items = $state<any[]>([]);
  let loading = $state(true);
  let deleteId = $state<string | null>(null);
  let showConfirm = $state(false);

  async function load() {
    loading = true;
    try {
      const res = await categoriesApi.list({ limit: 100 });
      items = res.data ?? [];
    } catch (e) {
      notifications.error('Kategorien konnten nicht geladen werden.');
    } finally {
      loading = false;
    }
  }

  onMount(load);

  function getParentName(parentId: string | null) {
    if (!parentId) return '—';
    const parent = items.find(i => i.id === parentId);
    return parent?.translations?.[0]?.name ?? parentId;
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
      notifications.success('Kategorie gelöscht.');
      showConfirm = false;
      deleteId = null;
      load();
    } catch (e) {
      notifications.error('Löschen fehlgeschlagen.');
    }
  }
</script>

<div class="flex items-center justify-between mb-6">
  <h1 class="text-2xl font-bold text-gray-900">Kategorien</h1>
  <a href="{base}/categories/new" class="btn btn-primary">+ Neu</a>
</div>

<div class="card p-6">
  {#if loading}
    <div class="flex items-center justify-center h-32">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
    </div>
  {:else}
    <div class="overflow-x-auto">
      <table class="min-w-full divide-y divide-gray-200">
        <thead>
          <tr>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Name</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Slug</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Eltern</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Aktiv</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Position</th>
            <th class="px-4 py-3"></th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          {#each items as item}
            <tr class="hover:bg-gray-50 cursor-pointer" onclick={() => goto(`${base}/categories/${item.id}`)}>
              <td class="px-4 py-3 text-sm font-medium text-gray-900">{item.translations?.[0]?.name ?? item.id}</td>
              <td class="px-4 py-3 text-sm text-gray-600">{item.translations?.[0]?.slug ?? ''}</td>
              <td class="px-4 py-3 text-sm text-gray-500">{getParentName(item.parent_id)}</td>
              <td class="px-4 py-3 text-sm">
                {#if item.active}
                  <span class="badge badge-green">Aktiv</span>
                {:else}
                  <span class="badge badge-gray">Inaktiv</span>
                {/if}
              </td>
              <td class="px-4 py-3 text-sm text-gray-500">{item.position ?? 0}</td>
              <td class="px-4 py-3 text-right">
                <button class="btn btn-danger btn-sm" onclick={(e) => confirmDelete(item.id, e)}>Löschen</button>
              </td>
            </tr>
          {/each}
          {#if items.length === 0}
            <tr>
              <td colspan="6" class="px-4 py-6 text-center text-gray-400 text-sm">Keine Kategorien gefunden.</td>
            </tr>
          {/if}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<ConfirmModal
  open={showConfirm}
  title="Kategorie löschen"
  message="Soll diese Kategorie wirklich gelöscht werden?"
  onConfirm={doDelete}
  onCancel={() => { showConfirm = false; deleteId = null; }}
/>
