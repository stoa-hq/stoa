<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { t, locale } from 'svelte-i18n';
  import { shippingApi } from '$lib/api/shipping';
  import { tr } from '$lib/i18n/entity';
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
      const res = await shippingApi.list({ limit: 100 });
      items = res.data ?? [];
    } catch (e) {
      notifications.error($t('shipping.loadFailed'));
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
      await shippingApi.delete(deleteId);
      notifications.success($t('shipping.deleted'));
      showConfirm = false;
      deleteId = null;
      load();
    } catch (e) {
      notifications.error($t('common.deleteFailed'));
    }
  }
</script>

<div class="flex items-center justify-between mb-6">
  <h1 class="text-2xl font-bold text-[var(--text)]">{$t('shipping.title')}</h1>
  <a href="{base}/shipping/new" class="btn btn-primary">{$t('common.new')}</a>
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
            <th class="table-header">{$t('common.price')}</th>
            <th class="table-header">{$t('common.active')}</th>
            <th class="table-header"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-[var(--card-border)]">
          {#each items as item}
            <tr class="table-row cursor-pointer" onclick={() => goto(`${base}/shipping/${item.id}`)}>
              <td class="table-cell font-medium text-[var(--text)]">{tr(item.translations, 'name', $locale) || item.id}</td>
              <td class="table-cell text-[var(--text-muted)] tabular-nums">{$fmt.price(item.price_gross)}</td>
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
              <td colspan="4" class="table-cell text-center text-[var(--text-muted)] py-6">{$t('shipping.noShippingMethods')}</td>
            </tr>
          {/if}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<ConfirmModal
  open={showConfirm}
  title={$t('shipping.deleteTitle')}
  message={$t('shipping.deleteMessage')}
  onConfirm={doDelete}
  onCancel={() => { showConfirm = false; deleteId = null; }}
/>
