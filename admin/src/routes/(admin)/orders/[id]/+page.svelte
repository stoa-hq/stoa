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
  import Skeleton from '$lib/components/Skeleton.svelte';
  import PluginSlot from '$lib/components/PluginSlot.svelte';
  import { ArrowLeft, CreditCard, Copy, Check, ExternalLink } from 'lucide-svelte';

  let id = $derived($page.params.id as string);
  let loading = $state(true);
  let order = $state<any>(null);
  let transactions = $state<any[]>([]);
  let txLoading = $state(true);
  let showStatusModal = $state(false);
  let statusSubmitting = $state(false);

  let statusForm = $state({
    status: '',
    comment: '',
  });

  let guestTokenCopied = $state(false);
  let stripeDashboardBase = $state<string | null>(null);

  async function fetchStripeDashboardBase() {
    try {
      const token = localStorage.getItem('stoa_access_token');
      const res = await fetch('/plugins/stripe/health', {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      });
      if (!res.ok) return;
      const data = await res.json();
      const pk = data.publishable_key ?? '';
      const prefix = 'https://dashboard.stripe.com';
      stripeDashboardBase = pk.startsWith('pk_test_') ? `${prefix}/test/payments/` : `${prefix}/payments/`;
    } catch {
      // Stripe plugin not installed — graceful degradation
    }
  }

  async function copyGuestToken() {
    if (!order?.guest_token) return;
    await navigator.clipboard.writeText(order.guest_token);
    guestTokenCopied = true;
    setTimeout(() => guestTokenCopied = false, 2000);
  }

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

  function txStatusBadge(status: string): string {
    const map: Record<string, string> = {
      pending: 'badge-yellow',
      completed: 'badge-green',
      succeeded: 'badge-green',
      failed: 'badge-red',
      refunded: 'badge-gray',
      cancelled: 'badge-red',
    };
    return map[status] ?? 'badge-blue';
  }

  onMount(async () => {
    try {
      order = (await ordersApi.get(id)).data;
      statusForm.status = order.status ?? '';
    } catch (e) {
      notifications.error($t('orders.loadOneFailed'));
    } finally {
      loading = false;
    }

    try {
      const res = await ordersApi.getTransactions(id);
      transactions = res.data ?? [];
    } catch {
      // transactions not available — not critical
    } finally {
      txLoading = false;
    }

    fetchStripeDashboardBase();
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
  <a href="{base}/orders" class="inline-flex items-center gap-1 text-sm text-primary-500 hover:text-primary-400 transition-colors">
    <ArrowLeft class="w-4 h-4" /> {$t('common.back')}
  </a>
</div>

{#if loading}
  <div class="space-y-6">
    <div class="card p-6">
      <Skeleton height="h-6" class="w-48 mb-4" />
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {#each Array(6) as _}
          <div class="space-y-2">
            <Skeleton height="h-3" class="w-20" />
            <Skeleton height="h-5" class="w-32" />
          </div>
        {/each}
      </div>
    </div>
  </div>
{:else if order}
  <!-- Order Info Card -->
  <div class="card p-6 mb-6">
    <div class="flex items-center justify-between mb-4">
      <h1 class="text-xl font-bold text-[var(--text)]">{$t('orders.orderNumber')} #{order.order_number ?? order.id}</h1>
      {#if statusOptions.length > 0}
        <button class="btn btn-secondary btn-sm" onclick={() => { statusForm.status = statusOptions[0].value; showStatusModal = true; }}>
          {$t('orders.changeStatus')}
        </button>
      {/if}
    </div>

    <div class="grid grid-cols-2 sm:grid-cols-3 gap-4">
      <div>
        <p class="text-xs text-[var(--text-muted)] uppercase font-medium">{$t('common.status')}</p>
        <span class="badge {orderStatusBadge(order.status)} mt-1">{order.status}</span>
      </div>
      <div>
        <p class="text-xs text-[var(--text-muted)] uppercase font-medium">{$t('common.createdAt')}</p>
        <p class="text-sm text-[var(--text)] mt-1">{$fmt.dateTime(order.created_at)}</p>
      </div>
      <div>
        <p class="text-xs text-[var(--text-muted)] uppercase font-medium">{$t('common.updatedAt')}</p>
        <p class="text-sm text-[var(--text)] mt-1">{$fmt.dateTime(order.updated_at)}</p>
      </div>
      <div>
        <p class="text-xs text-[var(--text-muted)] uppercase font-medium">{$t('orders.subtotal')}</p>
        <p class="text-sm text-[var(--text)] mt-1 tabular-nums">{$fmt.price(order.subtotal_gross)}</p>
      </div>
      <div>
        <p class="text-xs text-[var(--text-muted)] uppercase font-medium">{$t('orders.shippingCost')}</p>
        <p class="text-sm text-[var(--text)] mt-1 tabular-nums">{$fmt.price(order.shipping_cost)}</p>
      </div>
      <div>
        <p class="text-xs text-[var(--text-muted)] uppercase font-medium">{$t('orders.total')}</p>
        <p class="text-sm font-bold text-[var(--text)] mt-1 tabular-nums">{$fmt.price(order.total)}</p>
      </div>
    </div>
  </div>

  <!-- Addresses -->
  {#if order.shipping_address || order.billing_address}
  <div class="grid grid-cols-1 sm:grid-cols-2 gap-6 mb-6">
    <div class="card p-6">
      <h2 class="text-sm font-semibold text-[var(--text-muted)] uppercase tracking-wide mb-3">{$t('orders.shippingAddress')}</h2>
      {#if order.shipping_address}
        <address class="not-italic text-sm text-[var(--text)] space-y-1">
          <p class="font-semibold">{order.shipping_address.first_name ?? ''} {order.shipping_address.last_name ?? ''}</p>
          {#if order.shipping_address.company}<p class="text-[var(--text-muted)]">{order.shipping_address.company}</p>{/if}
          <p>{order.shipping_address.street ?? ''}</p>
          <p>{order.shipping_address.zip ?? ''} {order.shipping_address.city ?? ''}</p>
          <p class="uppercase tracking-wider text-xs text-[var(--text-muted)]">{order.shipping_address.country_code ?? ''}</p>
          {#if order.shipping_address.email}<p class="text-primary-500 text-xs mt-2">{order.shipping_address.email}</p>{/if}
          {#if order.shipping_address.phone}<p class="text-[var(--text-muted)] text-xs">{order.shipping_address.phone}</p>{/if}
        </address>
      {:else}
        <p class="text-sm text-[var(--text-muted)] italic">{$t('orders.noShippingAddress')}</p>
      {/if}
    </div>
    <div class="card p-6">
      <h2 class="text-sm font-semibold text-[var(--text-muted)] uppercase tracking-wide mb-3">{$t('orders.billingAddress')}</h2>
      {#if order.billing_address}
        <address class="not-italic text-sm text-[var(--text)] space-y-1">
          <p class="font-semibold">{order.billing_address.first_name ?? ''} {order.billing_address.last_name ?? ''}</p>
          {#if order.billing_address.company}<p class="text-[var(--text-muted)]">{order.billing_address.company}</p>{/if}
          <p>{order.billing_address.street ?? ''}</p>
          <p>{order.billing_address.zip ?? ''} {order.billing_address.city ?? ''}</p>
          <p class="uppercase tracking-wider text-xs text-[var(--text-muted)]">{order.billing_address.country_code ?? ''}</p>
          {#if order.billing_address.email}<p class="text-primary-500 text-xs mt-2">{order.billing_address.email}</p>{/if}
          {#if order.billing_address.phone}<p class="text-[var(--text-muted)] text-xs">{order.billing_address.phone}</p>{/if}
        </address>
      {:else}
        <p class="text-sm text-[var(--text-muted)] italic">{$t('orders.noBillingAddress')}</p>
      {/if}
    </div>
  </div>
  {/if}

  <!-- Order Items -->
  <div class="card p-6 mb-6">
    <h2 class="text-lg font-semibold text-[var(--text)] mb-4">{$t('orders.items')}</h2>
    <div class="overflow-x-auto">
      <table class="min-w-full divide-y divide-[var(--card-border)]">
        <thead>
          <tr>
            <th class="table-header">{$t('orders.product')}</th>
            <th class="table-header">{$t('products.sku')}</th>
            <th class="table-header">{$t('orders.quantity')}</th>
            <th class="table-header">{$t('orders.unitPrice')}</th>
            <th class="table-header">{$t('orders.total')}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-[var(--card-border)]">
          {#each order.items ?? [] as item}
            <tr class="table-row">
              <td class="table-cell text-[var(--text)]">{item.name ?? '—'}</td>
              <td class="table-cell text-[var(--text-muted)] font-mono text-xs">{item.sku ?? '—'}</td>
              <td class="table-cell text-[var(--text-muted)] tabular-nums">{item.quantity}</td>
              <td class="table-cell text-[var(--text-muted)] tabular-nums">{$fmt.price(item.unit_price_gross)}</td>
              <td class="table-cell text-[var(--text-muted)] tabular-nums">{$fmt.price(item.total_gross)}</td>
            </tr>
          {/each}
          {#if (order.items ?? []).length === 0}
            <tr>
              <td colspan="5" class="table-cell text-center text-[var(--text-muted)] py-4">{$t('orders.noItems')}</td>
            </tr>
          {/if}
        </tbody>
      </table>
    </div>
  </div>

  <!-- Payment Transactions -->
  <div class="card p-6 mb-6">
    <div class="flex items-center gap-2 mb-4">
      <CreditCard class="w-5 h-5 text-[var(--text-muted)]" />
      <h2 class="text-lg font-semibold text-[var(--text)]">{$t('orders.transactions')}</h2>
    </div>
    {#if order.guest_token}
      <div class="mb-4 flex items-center gap-3 rounded-lg bg-primary-600/5 dark:bg-primary-400/5 border border-primary-600/10 dark:border-primary-400/10 px-4 py-3">
        <div class="flex-1 min-w-0">
          <p class="text-xs font-medium text-[var(--text-muted)] uppercase tracking-wide mb-1">{$t('orders.guestSession')}</p>
          <code class="text-sm font-mono text-[var(--text)] select-all break-all">{order.guest_token}</code>
        </div>
        <button
          type="button"
          class="shrink-0 p-1.5 rounded-md text-[var(--text-muted)] hover:text-[var(--text)] hover:bg-gray-100 dark:hover:bg-white/5 transition-colors"
          onclick={copyGuestToken}
          title={$t('orders.copyGuestToken')}
        >
          {#if guestTokenCopied}
            <Check class="w-4 h-4 text-green-500" />
          {:else}
            <Copy class="w-4 h-4" />
          {/if}
        </button>
      </div>
    {/if}
    {#if txLoading}
      <div class="space-y-3">
        {#each Array(2) as _}
          <Skeleton height="h-10" />
        {/each}
      </div>
    {:else if transactions.length === 0}
      <p class="text-sm text-[var(--text-muted)]">{$t('orders.noTransactions')}</p>
    {:else}
      <div class="overflow-x-auto">
        <table class="min-w-full divide-y divide-[var(--card-border)]">
          <thead>
            <tr>
              <th class="table-header">{$t('common.status')}</th>
              <th class="table-header">{$t('orders.amount')}</th>
              <th class="table-header">{$t('orders.providerReference')}</th>
              <th class="table-header">{$t('common.createdAt')}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-[var(--card-border)]">
            {#each transactions as tx}
              <tr class="table-row">
                <td class="table-cell">
                  <span class="badge {txStatusBadge(tx.status)}">{tx.status}</span>
                </td>
                <td class="table-cell text-[var(--text)] tabular-nums font-medium">{$fmt.price(tx.amount)}</td>
                <td class="table-cell text-[var(--text-muted)] font-mono text-xs">
                  {#if tx.provider_reference && stripeDashboardBase && tx.provider_reference.startsWith('pi_')}
                    <a
                      href="{stripeDashboardBase}{tx.provider_reference}"
                      target="_blank"
                      rel="noopener noreferrer"
                      class="inline-flex items-center gap-1 text-primary-500 hover:text-primary-400 transition-colors"
                      title={$t('orders.viewInStripe')}
                    >
                      {tx.provider_reference}
                      <ExternalLink class="w-3 h-3" />
                    </a>
                  {:else}
                    {tx.provider_reference || '—'}
                  {/if}
                </td>
                <td class="table-cell text-[var(--text-muted)]">{$fmt.dateTime(tx.created_at)}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>

  <!-- Plugin Slot for Payment Extensions -->
  <PluginSlot slot="admin:order:payment" context={{ orderId: id, order }} />
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
