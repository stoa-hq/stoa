<script lang="ts">
  import { onMount } from 'svelte';
  import { base } from '$app/paths';
  import { page } from '$app/stores';
  import { t } from 'svelte-i18n';
  import { customersApi } from '$lib/api/customers';
  import { ordersApi } from '$lib/api/orders';
  import { notifications } from '$lib/stores/notifications';
  import { fmt } from '$lib/i18n/formatters';
  import { orderStatusBadge } from '$lib/utils';

  let id = $derived($page.params.id as string);
  let loading = $state(true);
  let submitting = $state(false);
  let orders = $state<any[]>([]);

  let form = $state({
    email: '',
    first_name: '',
    last_name: '',
    active: true,
  });

  onMount(async () => {
    try {
      const [customer, ordersRes] = await Promise.all([
        customersApi.get(id),
        ordersApi.list({ customer_id: id, limit: 10 }).catch(() => ({ data: [] })),
      ]);
      form = {
        email: customer.data.email ?? '',
        first_name: customer.data.first_name ?? '',
        last_name: customer.data.last_name ?? '',
        active: customer.data.active ?? true,
      };
      orders = (ordersRes as any).data ?? [];
    } catch (e) {
      notifications.error($t('customers.loadOneFailed'));
    } finally {
      loading = false;
    }
  });

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    submitting = true;
    try {
      await customersApi.update(id, form);
      notifications.success($t('customers.saved'));
    } catch (e) {
      notifications.error($t('common.saveFailed'));
    } finally {
      submitting = false;
    }
  }
</script>

<div class="mb-6">
  <a href="{base}/customers" class="text-sm text-primary-500 hover:text-primary-400 transition-colors">&larr; {$t('common.back')}</a>
</div>

{#if loading}
  <div class="flex items-center justify-center h-32">
    <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
  </div>
{:else}
  <div class="card p-6 max-w-2xl mb-6">
    <h1 class="text-xl font-bold text-[var(--text)] mb-6">{$t('customers.editCustomer')}</h1>

    <form onsubmit={handleSubmit} class="space-y-4">
      <div>
        <label class="label" for="email">{$t('common.email')}</label>
        <input id="email" class="input" type="email" bind:value={form.email} />
      </div>
      <div>
        <label class="label" for="first_name">{$t('customers.firstName')}</label>
        <input id="first_name" class="input" type="text" bind:value={form.first_name} />
      </div>
      <div>
        <label class="label" for="last_name">{$t('customers.lastName')}</label>
        <input id="last_name" class="input" type="text" bind:value={form.last_name} />
      </div>
      <div class="flex items-center gap-2">
        <input id="active" type="checkbox" bind:checked={form.active} class="h-4 w-4 rounded border-gray-300 text-primary-600" />
        <label for="active" class="text-sm text-[var(--text-muted)]">{$t('common.active')}</label>
      </div>
      <div class="flex gap-3 pt-2">
        <button type="submit" class="btn btn-primary" disabled={submitting}>
          {submitting ? $t('common.saving') : $t('common.save')}
        </button>
        <a href="{base}/customers" class="btn btn-secondary">{$t('common.cancel')}</a>
      </div>
    </form>
  </div>

  <!-- Orders -->
  <div class="card p-6 max-w-2xl">
    <h2 class="text-lg font-semibold text-[var(--text)] mb-4">{$t('customers.orders')}</h2>
    {#if orders.length === 0}
      <p class="text-sm text-[var(--text-muted)]">{$t('customers.noOrders')}</p>
    {:else}
      <div class="overflow-x-auto">
        <table class="min-w-full divide-y divide-[var(--card-border)]">
          <thead>
            <tr>
              <th class="table-header">{$t('orders.orderNumber')}</th>
              <th class="table-header">{$t('common.status')}</th>
              <th class="table-header">{$t('orders.total')}</th>
              <th class="table-header">{$t('common.createdAt')}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-[var(--card-border)]">
            {#each orders as order}
              {@const badgeClass = orderStatusBadge(order.status)}
              <tr>
                <td class="px-4 py-2 text-sm text-[var(--text)]">
                  <a href="{base}/orders/{order.id}" class="text-primary-500 hover:text-primary-400 transition-colors">
                    #{order.order_number ?? order.id}
                  </a>
                </td>
                <td class="px-4 py-2 text-sm">
                  <span class="badge {badgeClass}">{order.status}</span>
                </td>
                <td class="px-4 py-2 text-sm text-[var(--text-muted)]">{$fmt.price(order.total)}</td>
                <td class="px-4 py-2 text-sm text-[var(--text-muted)]">{$fmt.dateTime(order.created_at)}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
{/if}
