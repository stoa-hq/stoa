<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { ordersApi } from '$lib/api/orders';
  import { productsApi } from '$lib/api/products';
  import { customersApi } from '$lib/api/customers';
  import { notifications } from '$lib/stores/notifications';
  import { orderStatusBadge } from '$lib/utils';
  import { t } from 'svelte-i18n';
  import { fmt } from '$lib/i18n/formatters';
  import { ShoppingCart, Users, Package, TrendingUp } from 'lucide-svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';

  let loading = $state(true);
  let stats = $state({
    totalOrders: 0,
    totalCustomers: 0,
    totalProducts: 0,
    totalRevenue: 0,
  });
  let recentOrders = $state<any[]>([]);

  const kpiCards = $derived([
    { key: 'totalOrders', label: 'dashboard.totalOrders', value: stats.totalOrders, icon: ShoppingCart, color: 'text-blue-500 dark:text-blue-400', bg: 'bg-blue-50 dark:bg-blue-900/20' },
    { key: 'totalCustomers', label: 'dashboard.totalCustomers', value: stats.totalCustomers, icon: Users, color: 'text-green-500 dark:text-green-400', bg: 'bg-green-50 dark:bg-green-900/20' },
    { key: 'totalProducts', label: 'dashboard.totalProducts', value: stats.totalProducts, icon: Package, color: 'text-primary-500 dark:text-primary-400', bg: 'bg-primary-50 dark:bg-primary-900/20' },
    { key: 'totalRevenue', label: 'dashboard.revenueLastOrders', value: $fmt.price(stats.totalRevenue), icon: TrendingUp, color: 'text-amber-500 dark:text-amber-400', bg: 'bg-amber-50 dark:bg-amber-900/20', isFormatted: true },
  ]);

  onMount(async () => {
    try {
      const [ordersRes, productsRes, customersRes] = await Promise.all([
        ordersApi.list({ limit: 5, sort: 'created_at', order: 'desc' }),
        productsApi.list({ limit: 1 }),
        customersApi.list({ limit: 1 }),
      ]);

      recentOrders = ordersRes.data ?? [];

      const totalRevenue = (ordersRes.data ?? []).reduce(
        (sum: number, o: any) => sum + (o.total ?? 0),
        0
      );

      stats = {
        totalOrders: ordersRes.meta?.total ?? 0,
        totalProducts: productsRes.meta?.total ?? 0,
        totalCustomers: customersRes.meta?.total ?? 0,
        totalRevenue,
      };
    } catch (e) {
      notifications.error($t('dashboard.loadFailed'));
    } finally {
      loading = false;
    }
  });
</script>

<div class="space-y-6">
  <!-- KPI Cards -->
  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
    {#if loading}
      {#each Array(4) as _}
        <div class="card p-6">
          <div class="flex items-center gap-3">
            <div class="skeleton w-10 h-10 rounded-xl"></div>
            <div class="flex-1 space-y-2">
              <Skeleton height="h-3" class="w-24" />
              <Skeleton height="h-6" class="w-16" />
            </div>
          </div>
        </div>
      {/each}
    {:else}
      {#each kpiCards as card}
        <div class="card p-6 hover:scale-[1.02] transition-transform duration-150 cursor-default">
          <div class="flex items-center gap-3">
            <div class="w-10 h-10 rounded-xl {card.bg} flex items-center justify-center">
              <card.icon class="w-5 h-5 {card.color}" />
            </div>
            <div>
              <p class="text-xs text-[var(--text-muted)]">{$t(card.label)}</p>
              <p class="text-2xl font-bold text-[var(--text)] tabular-nums mt-0.5">
                {card.isFormatted ? card.value : card.value}
              </p>
            </div>
          </div>
        </div>
      {/each}
    {/if}
  </div>

  <!-- Recent Orders -->
  <div class="card p-6">
    <h2 class="text-lg font-semibold text-[var(--text)] mb-4">{$t('dashboard.recentOrders')}</h2>
    {#if loading}
      <div class="space-y-3">
        {#each Array(5) as _}
          <Skeleton height="h-10" />
        {/each}
      </div>
    {:else if recentOrders.length === 0}
      <p class="text-[var(--text-muted)] text-sm">{$t('dashboard.noOrders')}</p>
    {:else}
      <!-- Desktop Table -->
      <div class="hidden sm:block overflow-x-auto">
        <table class="min-w-full divide-y divide-[var(--card-border)]">
          <thead>
            <tr>
              <th class="table-header">{$t('dashboard.orderNumber')}</th>
              <th class="table-header">{$t('common.status')}</th>
              <th class="table-header">{$t('dashboard.total')}</th>
              <th class="table-header">{$t('common.createdAt')}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-[var(--card-border)]">
            {#each recentOrders as order}
              {@const badgeClass = orderStatusBadge(order.status)}
              <tr class="table-row cursor-pointer" onclick={() => goto(`${base}/orders/${order.id}`)}>
                <td class="table-cell font-medium text-[var(--text)]">#{order.order_number ?? order.id}</td>
                <td class="table-cell">
                  <span class="badge {badgeClass}">{order.status}</span>
                </td>
                <td class="table-cell text-[var(--text-muted)] tabular-nums">{$fmt.price(order.total)}</td>
                <td class="table-cell text-[var(--text-muted)]">{$fmt.dateTime(order.created_at)}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
      <!-- Mobile Cards -->
      <div class="sm:hidden space-y-3">
        {#each recentOrders as order}
          {@const badgeClass = orderStatusBadge(order.status)}
          <div
            class="p-3 rounded-lg bg-[var(--surface)] border border-[var(--card-border)] cursor-pointer hover:bg-gray-50 dark:hover:bg-white/5 transition-colors"
            role="button"
            tabindex="0"
            onclick={() => goto(`${base}/orders/${order.id}`)}
            onkeydown={(e) => e.key === 'Enter' && goto(`${base}/orders/${order.id}`)}
          >
            <div class="flex items-center justify-between mb-1">
              <span class="font-medium text-sm text-[var(--text)]">#{order.order_number ?? order.id}</span>
              <span class="badge {badgeClass}">{order.status}</span>
            </div>
            <div class="flex items-center justify-between text-xs text-[var(--text-muted)]">
              <span class="tabular-nums">{$fmt.price(order.total)}</span>
              <span>{$fmt.dateTime(order.created_at)}</span>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>
