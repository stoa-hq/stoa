<script lang="ts">
  import { onMount } from 'svelte';
  import { ordersApi } from '$lib/api/orders';
  import { productsApi } from '$lib/api/products';
  import { customersApi } from '$lib/api/customers';
  import { notifications } from '$lib/stores/notifications';
  import { formatPrice, formatDateTime, orderStatusBadge } from '$lib/utils';

  let loading = $state(true);
  let stats = $state({
    totalOrders: 0,
    totalCustomers: 0,
    totalProducts: 0,
    totalRevenue: 0,
  });
  let recentOrders = $state<any[]>([]);

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
      notifications.error('Dashboard-Daten konnten nicht geladen werden.');
    } finally {
      loading = false;
    }
  });
</script>

{#if loading}
  <div class="flex items-center justify-center h-64">
    <div class="animate-spin rounded-full h-10 w-10 border-b-2 border-primary-600"></div>
  </div>
{:else}
  <div class="space-y-6">
    <!-- Stats Cards -->
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      <div class="card p-6">
        <p class="text-sm text-gray-500">Bestellungen gesamt</p>
        <p class="text-3xl font-bold text-gray-900 mt-1">{stats.totalOrders}</p>
      </div>
      <div class="card p-6">
        <p class="text-sm text-gray-500">Kunden gesamt</p>
        <p class="text-3xl font-bold text-gray-900 mt-1">{stats.totalCustomers}</p>
      </div>
      <div class="card p-6">
        <p class="text-sm text-gray-500">Produkte gesamt</p>
        <p class="text-3xl font-bold text-gray-900 mt-1">{stats.totalProducts}</p>
      </div>
      <div class="card p-6">
        <p class="text-sm text-gray-500">Umsatz (letzte 5 Bestellungen)</p>
        <p class="text-3xl font-bold text-gray-900 mt-1">{formatPrice(stats.totalRevenue)}</p>
      </div>
    </div>

    <!-- Recent Orders -->
    <div class="card p-6">
      <h2 class="text-lg font-semibold text-gray-900 mb-4">Letzte Bestellungen</h2>
      {#if recentOrders.length === 0}
        <p class="text-gray-500 text-sm">Keine Bestellungen vorhanden.</p>
      {:else}
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200">
            <thead>
              <tr>
                <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Bestellnr.</th>
                <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Gesamt</th>
                <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Erstellt</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-gray-200">
              {#each recentOrders as order}
                {@const badgeClass = orderStatusBadge(order.status)}
                <tr class="hover:bg-gray-50">
                  <td class="px-4 py-3 text-sm font-medium text-gray-900">#{order.order_number ?? order.id}</td>
                  <td class="px-4 py-3 text-sm">
                    <span class="badge {badgeClass}">{order.status}</span>
                  </td>
                  <td class="px-4 py-3 text-sm text-gray-700">{formatPrice(order.total)}</td>
                  <td class="px-4 py-3 text-sm text-gray-500">{formatDateTime(order.created_at)}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    </div>
  </div>
{/if}
