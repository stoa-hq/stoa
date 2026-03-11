<script lang="ts">
  import { onMount } from 'svelte';
  import { base } from '$app/paths';
  import { page } from '$app/stores';
  import { t } from 'svelte-i18n';
  import { ordersApi } from '$lib/api/orders';
  import { notifications } from '$lib/stores/notifications';
  import { fmt } from '$lib/i18n/formatters';
  import { orderStatusBadge } from '$lib/utils';
  import Modal from '$lib/components/Modal.svelte';

  let id = $derived($page.params.id as string);
  let loading = $state(true);
  let order = $state<any>(null);
  let showStatusModal = $state(false);
  let statusSubmitting = $state(false);

  let statusForm = $state({
    status: '',
    comment: '',
  });

  const allStatusKeys: Record<string, string> = {
    pending: 'orders.pending',
    confirmed: 'orders.confirmed',
    processing: 'orders.processing',
    shipped: 'orders.shipped',
    delivered: 'orders.delivered',
    cancelled: 'orders.cancelled',
    refunded: 'orders.refunded',
  };

  const validTransitions: Record<string, string[]> = {
    pending: ['confirmed', 'cancelled'],
    confirmed: ['processing', 'cancelled'],
    processing: ['shipped', 'cancelled'],
    shipped: ['delivered'],
    delivered: ['refunded'],
    cancelled: [],
    refunded: [],
  };

  const statusOptions = $derived(
    (validTransitions[order?.status ?? ''] ?? []).map((v) => ({ value: v, key: allStatusKeys[v] }))
  );

  onMount(async () => {
    try {
      order = (await ordersApi.get(id)).data;
      statusForm.status = order.status ?? '';
    } catch (e) {
      notifications.error($t('orders.loadOneFailed'));
    } finally {
      loading = false;
    }
  });

  async function handleStatusSubmit(e: SubmitEvent) {
    e.preventDefault();
    statusSubmitting = true;
    try {
      await ordersApi.updateStatus(id, statusForm.status, statusForm.comment || undefined);
      notifications.success($t('orders.statusUpdated'));
      order = (await ordersApi.get(id)).data;
      showStatusModal = false;
    } catch (e) {
      notifications.error($t('orders.statusChangeFailed'));
    } finally {
      statusSubmitting = false;
    }
  }
</script>

<div class="mb-6">
  <a href="{base}/orders" class="text-sm text-primary-600 hover:underline">&larr; {$t('common.back')}</a>
</div>

{#if loading}
  <div class="flex items-center justify-center h-32">
    <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
  </div>
{:else if order}
  <!-- Order Info Card -->
  <div class="card p-6 mb-6">
    <div class="flex items-center justify-between mb-4">
      <h1 class="text-xl font-bold text-gray-900">{$t('orders.orderNumber')} #{order.order_number ?? order.id}</h1>
      {#if statusOptions.length > 0}
        <button class="btn btn-secondary btn-sm" onclick={() => { statusForm.status = statusOptions[0].value; showStatusModal = true; }}>
          {$t('orders.changeStatus')}
        </button>
      {/if}
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
      <div>
        <p class="text-xs text-gray-500 uppercase font-medium">{$t('common.status')}</p>
        <span class="badge {orderStatusBadge(order.status)} mt-1">{order.status}</span>
      </div>
      <div>
        <p class="text-xs text-gray-500 uppercase font-medium">{$t('common.createdAt')}</p>
        <p class="text-sm text-gray-900 mt-1">{$fmt.dateTime(order.created_at)}</p>
      </div>
      <div>
        <p class="text-xs text-gray-500 uppercase font-medium">{$t('common.updatedAt')}</p>
        <p class="text-sm text-gray-900 mt-1">{$fmt.dateTime(order.updated_at)}</p>
      </div>
      <div>
        <p class="text-xs text-gray-500 uppercase font-medium">{$t('orders.subtotal')}</p>
        <p class="text-sm text-gray-900 mt-1">{$fmt.price(order.subtotal_gross)}</p>
      </div>
      <div>
        <p class="text-xs text-gray-500 uppercase font-medium">{$t('orders.shippingCost')}</p>
        <p class="text-sm text-gray-900 mt-1">{$fmt.price(order.shipping_cost)}</p>
      </div>
      <div>
        <p class="text-xs text-gray-500 uppercase font-medium">{$t('orders.total')}</p>
        <p class="text-sm font-bold text-gray-900 mt-1">{$fmt.price(order.total)}</p>
      </div>
    </div>
  </div>

  <!-- Addresses -->
  {#if order.shipping_address || order.billing_address}
  <div class="grid grid-cols-1 sm:grid-cols-2 gap-6 mb-6">
    <div class="card p-6">
      <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-3">{$t('orders.shippingAddress')}</h2>
      {#if order.shipping_address}
        <address class="not-italic text-sm text-gray-800 space-y-1">
          <p class="font-semibold">{order.shipping_address.first_name ?? ''} {order.shipping_address.last_name ?? ''}</p>
          {#if order.shipping_address.company}<p class="text-gray-500">{order.shipping_address.company}</p>{/if}
          <p>{order.shipping_address.street ?? ''}</p>
          <p>{order.shipping_address.zip ?? ''} {order.shipping_address.city ?? ''}</p>
          <p class="uppercase tracking-wider text-xs text-gray-500">{order.shipping_address.country_code ?? ''}</p>
          {#if order.shipping_address.email}<p class="text-primary-600 text-xs mt-2">{order.shipping_address.email}</p>{/if}
          {#if order.shipping_address.phone}<p class="text-gray-500 text-xs">{order.shipping_address.phone}</p>{/if}
        </address>
      {:else}
        <p class="text-sm text-gray-400 italic">{$t('orders.noShippingAddress')}</p>
      {/if}
    </div>
    <div class="card p-6">
      <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-3">{$t('orders.billingAddress')}</h2>
      {#if order.billing_address}
        <address class="not-italic text-sm text-gray-800 space-y-1">
          <p class="font-semibold">{order.billing_address.first_name ?? ''} {order.billing_address.last_name ?? ''}</p>
          {#if order.billing_address.company}<p class="text-gray-500">{order.billing_address.company}</p>{/if}
          <p>{order.billing_address.street ?? ''}</p>
          <p>{order.billing_address.zip ?? ''} {order.billing_address.city ?? ''}</p>
          <p class="uppercase tracking-wider text-xs text-gray-500">{order.billing_address.country_code ?? ''}</p>
          {#if order.billing_address.email}<p class="text-primary-600 text-xs mt-2">{order.billing_address.email}</p>{/if}
          {#if order.billing_address.phone}<p class="text-gray-500 text-xs">{order.billing_address.phone}</p>{/if}
        </address>
      {:else}
        <p class="text-sm text-gray-400 italic">{$t('orders.noBillingAddress')}</p>
      {/if}
    </div>
  </div>
  {/if}

  <!-- Order Items -->
  <div class="card p-6">
    <h2 class="text-lg font-semibold text-gray-900 mb-4">{$t('orders.items')}</h2>
    <div class="overflow-x-auto">
      <table class="min-w-full divide-y divide-gray-200">
        <thead>
          <tr>
            <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">{$t('orders.product')}</th>
            <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">{$t('products.sku')}</th>
            <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">{$t('orders.quantity')}</th>
            <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">{$t('orders.unitPrice')}</th>
            <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">{$t('orders.total')}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-200">
          {#each order.items ?? [] as item}
            <tr>
              <td class="px-4 py-2 text-sm text-gray-900">{item.name ?? '—'}</td>
              <td class="px-4 py-2 text-sm text-gray-600">{item.sku ?? '—'}</td>
              <td class="px-4 py-2 text-sm text-gray-700">{item.quantity}</td>
              <td class="px-4 py-2 text-sm text-gray-700">{$fmt.price(item.unit_price_gross)}</td>
              <td class="px-4 py-2 text-sm text-gray-700">{$fmt.price(item.total_gross)}</td>
            </tr>
          {/each}
          {#if (order.items ?? []).length === 0}
            <tr>
              <td colspan="5" class="px-4 py-4 text-center text-gray-400 text-sm">{$t('orders.noItems')}</td>
            </tr>
          {/if}
        </tbody>
      </table>
    </div>
  </div>
{/if}

<Modal open={showStatusModal} title={$t('orders.changeStatus')} onClose={() => showStatusModal = false}>
  <form onsubmit={handleStatusSubmit} class="space-y-4">
    <div>
      <label class="label" for="status">{$t('common.status')}</label>
      <select id="status" class="input" bind:value={statusForm.status}>
        {#each statusOptions as opt}
          <option value={opt.value}>{$t(opt.key)}</option>
        {/each}
      </select>
    </div>
    <div>
      <label class="label" for="comment">{$t('orders.comment')}</label>
      <textarea id="comment" class="input" rows="3" bind:value={statusForm.comment}></textarea>
    </div>
    <div class="flex gap-3 pt-2">
      <button type="submit" class="btn btn-primary" disabled={statusSubmitting}>
        {statusSubmitting ? $t('common.saving') : $t('common.save')}
      </button>
      <button type="button" class="btn btn-secondary" onclick={() => showStatusModal = false}>{$t('common.cancel')}</button>
    </div>
  </form>
</Modal>
