<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { t } from 'svelte-i18n';
  import { discountsApi } from '$lib/api/discounts';
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
      const res = await discountsApi.list({ limit: 100 });
      items = res.data ?? [];
    } catch (e) {
      notifications.error($t('discounts.loadFailed'));
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
      await discountsApi.delete(deleteId);
      notifications.success($t('discounts.deleted'));
      showConfirm = false;
      deleteId = null;
      load();
    } catch (e) {
      notifications.error($t('common.deleteFailed'));
    }
  }

  function formatType(type: string) {
    return type === 'percentage' ? $t('discounts.percentage') : $t('discounts.fixed');
  }

  function formatValue(item: any) {
    if (item.type === 'percentage') return `${item.value / 100}%`;
    return `${(item.value / 100).toFixed(2)} €`;
  }
</script>

<div class="flex items-center justify-between mb-6">
  <h1 class="text-2xl font-bold text-[var(--text)]">{$t('discounts.title')}</h1>
  <a href="{base}/discounts/new" class="btn btn-primary">{$t('common.new')}</a>
</div>

<div class="card p-6">
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
            <th class="table-header">{$t('discounts.code')}</th>
            <th class="table-header">{$t('discounts.type')}</th>
            <th class="table-header">{$t('discounts.value')}</th>
            <th class="table-header">{$t('discounts.usages')}</th>
            <th class="table-header">{$t('discounts.validUntil')}</th>
            <th class="table-header">{$t('common.active')}</th>
            <th class="table-header"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-[var(--card-border)]">
          {#each items as item}
            <tr class="table-row cursor-pointer" onclick={() => goto(`${base}/discounts/${item.id}`)}>
              <td class="table-cell font-mono font-medium text-[var(--text)]">{item.code}</td>
              <td class="table-cell text-[var(--text-muted)]">{formatType(item.type)}</td>
              <td class="table-cell text-[var(--text-muted)] tabular-nums">{formatValue(item)}</td>
              <td class="table-cell text-[var(--text-muted)] tabular-nums">
                {item.used_count ?? 0}{item.max_uses ? ` / ${item.max_uses}` : ''}
              </td>
              <td class="table-cell text-[var(--text-muted)]">{item.valid_until ? $fmt.date(item.valid_until) : '—'}</td>
              <td class="table-cell">
                {#if item.active}
                  <span class="badge badge-green">{$t('common.active')}</span>
                {:else}
                  <span class="badge badge-gray">{$t('common.inactive')}</span>
                {/if}
              </td>
              <td class="table-cell text-right">
                <button class="btn btn-danger btn-sm" onclick={(e) => confirmDelete(item.id, e)}>{$t('common.delete')}</button>
              </td>
            </tr>
          {/each}
          {#if items.length === 0}
            <tr>
              <td colspan="7" class="table-cell text-center text-[var(--text-muted)] py-6">{$t('discounts.noDiscounts')}</td>
            </tr>
          {/if}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<ConfirmModal
  open={showConfirm}
  title={$t('discounts.deleteTitle')}
  message={$t('discounts.deleteMessage')}
  onConfirm={doDelete}
  onCancel={() => { showConfirm = false; deleteId = null; }}
/>
