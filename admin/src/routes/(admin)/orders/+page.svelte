<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { ordersApi } from '$lib/api/orders';
  import { notifications } from '$lib/stores/notifications';
  import { formatPrice, formatDateTime, orderStatusBadge } from '$lib/utils';
  import Pagination from '$lib/components/Pagination.svelte';

  let items = $state<any[]>([]);
  let meta = $state<any>(null);
  let currentPage = $state(1);
  let limit = $state(25);
  let loading = $state(true);
  let statusFilter = $state('');

  const statusOptions = [
    { value: '', label: 'Alle Status' },
    { value: 'pending', label: 'Ausstehend' },
    { value: 'processing', label: 'In Bearbeitung' },
    { value: 'shipped', label: 'Versendet' },
    { value: 'delivered', label: 'Geliefert' },
    { value: 'cancelled', label: 'Storniert' },
    { value: 'refunded', label: 'Erstattet' },
  ];

  async function load() {
    loading = true;
    try {
      const params: any = { page: currentPage, limit };
      if (statusFilter) params.status = statusFilter;
      const res = await ordersApi.list(params);
      items = res.data ?? [];
      meta = res.meta ?? null;
    } catch (e) {
      notifications.error('Bestellungen konnten nicht geladen werden.');
    } finally {
      loading = false;
    }
  }

  onMount(load);

  function handlePageChange(p: number) {
    currentPage = p;
    load();
  }

  function handleStatusFilter() {
    currentPage = 1;
    load();
  }
</script>

<div class="flex items-center justify-between mb-6">
  <h1 class="text-2xl font-bold text-gray-900">Bestellungen</h1>
</div>

<div class="card p-6">
  <div class="mb-4">
    <select class="input max-w-xs" bind:value={statusFilter} onchange={handleStatusFilter}>
      {#each statusOptions as opt}
        <option value={opt.value}>{opt.label}</option>
      {/each}
    </select>
  </div>

  {#if loading}
    <div class="flex items-center justify-center h-32">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
    </div>
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
          {#each items as item}
            {@const badgeClass = orderStatusBadge(item.status)}
            <tr class="hover:bg-gray-50 cursor-pointer" onclick={() => goto(`${base}/orders/${item.id}`)}>
              <td class="px-4 py-3 text-sm font-medium text-gray-900">#{item.order_number ?? item.id}</td>
              <td class="px-4 py-3 text-sm">
                <span class="badge {badgeClass}">{item.status}</span>
              </td>
              <td class="px-4 py-3 text-sm text-gray-700">{formatPrice(item.total)}</td>
              <td class="px-4 py-3 text-sm text-gray-500">{formatDateTime(item.created_at)}</td>
            </tr>
          {/each}
          {#if items.length === 0}
            <tr>
              <td colspan="4" class="px-4 py-6 text-center text-gray-400 text-sm">Keine Bestellungen gefunden.</td>
            </tr>
          {/if}
        </tbody>
      </table>
    </div>

    {#if meta}
      <div class="mt-4">
        <Pagination
          currentPage={currentPage}
          totalPages={Math.ceil(meta.total / limit)}
          onPageChange={handlePageChange}
        />
      </div>
    {/if}
  {/if}
</div>
