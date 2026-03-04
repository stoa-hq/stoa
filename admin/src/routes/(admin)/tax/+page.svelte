<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { taxApi } from '$lib/api/tax';
  import { notifications } from '$lib/stores/notifications';
  import { formatTaxRate } from '$lib/utils';
  import ConfirmModal from '$lib/components/ConfirmModal.svelte';

  let items = $state<any[]>([]);
  let loading = $state(true);
  let deleteId = $state<string | null>(null);
  let showConfirm = $state(false);

  async function load() {
    loading = true;
    try {
      const res = await taxApi.list({ limit: 100 });
      items = res.data ?? [];
    } catch (e) {
      notifications.error('Steuerregeln konnten nicht geladen werden.');
    } finally {
      loading = false;
    }
  }

  onMount(load);

  function confirmDelete(id: string, e: MouseEvent) {
    e.stopPropagation();
    deleteId = id;
    showConfirm = true;
  }

  async function doDelete() {
    if (!deleteId) return;
    try {
      await taxApi.delete(deleteId);
      notifications.success('Steuerregel gelöscht.');
      showConfirm = false;
      deleteId = null;
      load();
    } catch (e) {
      notifications.error('Löschen fehlgeschlagen.');
    }
  }

  function formatType(type: string) {
    const map: Record<string, string> = {
      standard: 'Standard',
      reduced: 'Ermäßigt',
      zero: 'Nullsatz',
    };
    return map[type] ?? type;
  }
</script>

<div class="flex items-center justify-between mb-6">
  <h1 class="text-2xl font-bold text-gray-900">Steuerregeln</h1>
  <a href="{base}/tax/new" class="btn btn-primary">+ Neu</a>
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
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Rate</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Land</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Typ</th>
            <th class="px-4 py-3"></th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          {#each items as item}
            <tr class="hover:bg-gray-50 cursor-pointer" onclick={() => goto(`${base}/tax/${item.id}`)}>
              <td class="px-4 py-3 text-sm font-medium text-gray-900">{item.name}</td>
              <td class="px-4 py-3 text-sm text-gray-700">{formatTaxRate(item.rate)}</td>
              <td class="px-4 py-3 text-sm text-gray-700">{item.country_code ?? '—'}</td>
              <td class="px-4 py-3 text-sm text-gray-700">{formatType(item.type)}</td>
              <td class="px-4 py-3 text-right">
                <button class="btn btn-danger btn-sm" onclick={(e) => confirmDelete(item.id, e)}>Löschen</button>
              </td>
            </tr>
          {/each}
          {#if items.length === 0}
            <tr>
              <td colspan="5" class="px-4 py-6 text-center text-gray-400 text-sm">Keine Steuerregeln gefunden.</td>
            </tr>
          {/if}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<ConfirmModal
  open={showConfirm}
  title="Steuerregel löschen"
  message="Soll diese Steuerregel wirklich gelöscht werden?"
  onConfirm={doDelete}
  onCancel={() => { showConfirm = false; deleteId = null; }}
/>
