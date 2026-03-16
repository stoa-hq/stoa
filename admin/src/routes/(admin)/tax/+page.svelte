<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { t } from 'svelte-i18n';
  import { taxApi } from '$lib/api/tax';
  import { notifications } from '$lib/stores/notifications';
  import { fmt } from '$lib/i18n/formatters';
  import ConfirmModal from '$lib/components/ConfirmModal.svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';

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
      notifications.error($t('tax.loadFailed'));
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
      notifications.success($t('tax.deleted'));
      showConfirm = false;
      deleteId = null;
      load();
    } catch (e) {
      notifications.error($t('common.deleteFailed'));
    }
  }

  function formatType(type: string) {
    const map: Record<string, string> = {
      standard: $t('tax.standard'),
      reduced: $t('tax.reduced'),
      zero: $t('tax.zero'),
    };
    return map[type] ?? type;
  }
</script>

<div class="flex items-center justify-between mb-6">
  <h1 class="text-2xl font-bold text-[var(--text)]">{$t('tax.title')}</h1>
  <a href="{base}/tax/new" class="btn btn-primary">{$t('common.new')}</a>
</div>

<div class="card p-6">
  {#if loading}
    <div class="space-y-3">
      {#each Array(3) as _}
        <Skeleton height="h-12" />
      {/each}
    </div>
  {:else}
    <div class="overflow-x-auto">
      <table class="min-w-full divide-y divide-[var(--card-border)]">
        <thead>
          <tr>
            <th class="table-header">{$t('common.name')}</th>
            <th class="table-header">{$t('tax.rate')}</th>
            <th class="table-header">{$t('tax.country')}</th>
            <th class="table-header">{$t('common.type')}</th>
            <th class="table-header"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-[var(--card-border)]">
          {#each items as item}
            <tr class="table-row cursor-pointer" onclick={() => goto(`${base}/tax/${item.id}`)}>
              <td class="table-cell font-medium text-[var(--text)]">{item.name}</td>
              <td class="table-cell text-[var(--text-muted)] tabular-nums">{$fmt.taxRate(item.rate)}</td>
              <td class="table-cell text-[var(--text-muted)]">{item.country_code ?? '—'}</td>
              <td class="table-cell text-[var(--text-muted)]">{formatType(item.type)}</td>
              <td class="table-cell text-right">
                <button class="btn btn-danger btn-sm" onclick={(e) => confirmDelete(item.id, e)}>{$t('common.delete')}</button>
              </td>
            </tr>
          {/each}
          {#if items.length === 0}
            <tr>
              <td colspan="5" class="table-cell text-center text-[var(--text-muted)] py-6">{$t('tax.noTaxRules')}</td>
            </tr>
          {/if}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<ConfirmModal
  open={showConfirm}
  title={$t('tax.deleteTitle')}
  message={$t('tax.deleteMessage')}
  onConfirm={doDelete}
  onCancel={() => { showConfirm = false; deleteId = null; }}
/>
