<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { t } from 'svelte-i18n';
  import { ordersApi } from '$lib/api/orders';
  import { notifications } from '$lib/stores/notifications';
  import { fmt } from '$lib/i18n/formatters';
  import { orderStatusBadge } from '$lib/utils';
  import Pagination from '$lib/components/Pagination.svelte';
  import SearchBar from '$lib/components/SearchBar.svelte';
  import FilterChips from '$lib/components/FilterChips.svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';

  let items = $state<any[]>([]);
  let meta = $state<any>(null);
  let currentPage = $state(1);
  let limit = $state(25);
  let loading = $state(true);
  let statusFilter = $state('');
  let search = $state('');

  const statusOptions = $derived([
    { value: '', label: $t('orders.allStatuses') },
    { value: 'pending', label: $t('orders.pending') },
    { value: 'processing', label: $t('orders.processing') },
    { value: 'shipped', label: $t('orders.shipped') },
    { value: 'delivered', label: $t('orders.delivered') },
    { value: 'cancelled', label: $t('orders.cancelled') },
    { value: 'refunded', label: $t('orders.refunded') },
  ]);

  async function load() {
    loading = true;
    try {
      const params: any = { page: currentPage, limit };
      if (statusFilter) params.status = statusFilter;
      if (search) params.search = search;
      const res = await ordersApi.list(params);
      items = res.data ?? [];
      meta = res.meta ?? null;
    } catch (e) {
      notifications.error($t('orders.loadFailed'));
    } finally {
      loading = false;
    }
  }

  onMount(load);

  function handlePageChange(p: number) {
    currentPage = p;
    load();
  }

  function handleStatusFilter(value: string) {
    statusFilter = value;
    currentPage = 1;
    load();
  }

  function handleSearch(value: string) {
    search = value;
    currentPage = 1;
    load();
  }
</script>

<div class="flex items-center justify-between mb-6">
  <h1 class="text-2xl font-bold text-[var(--text)]">{$t('orders.title')}</h1>
</div>

<div class="card p-6">
  <div class="flex flex-col sm:flex-row gap-3 mb-4">
    <SearchBar value={search} onSearch={handleSearch} placeholder={$t('orders.searchPlaceholder')} />
  </div>
  <div class="mb-4">
    <FilterChips options={statusOptions} selected={statusFilter} onSelect={handleStatusFilter} />
  </div>

  {#if loading}
    <div class="space-y-3">
      {#each Array(5) as _}
        <Skeleton height="h-12" />
      {/each}
    </div>
  {:else}
    <!-- Desktop Table -->
    <div class="hidden sm:block overflow-x-auto">
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
          {#each items as item}
            {@const badgeClass = orderStatusBadge(item.status)}
            <tr class="table-row cursor-pointer" onclick={() => goto(`${base}/orders/${item.id}`)}>
              <td class="table-cell font-medium text-[var(--text)]">#{item.order_number ?? item.id}</td>
              <td class="table-cell">
                <span class="badge {badgeClass}">{item.status}</span>
              </td>
              <td class="table-cell text-[var(--text-muted)] tabular-nums">{$fmt.price(item.total)}</td>
              <td class="table-cell text-[var(--text-muted)]">{$fmt.dateTime(item.created_at)}</td>
            </tr>
          {/each}
          {#if items.length === 0}
            <tr>
              <td colspan="4" class="table-cell text-center text-[var(--text-muted)] py-6">{$t('orders.noOrders')}</td>
            </tr>
          {/if}
        </tbody>
      </table>
    </div>
    <!-- Mobile Cards -->
    <div class="sm:hidden space-y-3">
      {#each items as item}
        {@const badgeClass = orderStatusBadge(item.status)}
        <div
          class="p-3 rounded-lg bg-[var(--surface)] border border-[var(--card-border)] cursor-pointer hover:bg-gray-50 dark:hover:bg-white/5 transition-colors"
          role="button" tabindex="0"
          onclick={() => goto(`${base}/orders/${item.id}`)}
          onkeydown={(e) => e.key === 'Enter' && goto(`${base}/orders/${item.id}`)}
        >
          <div class="flex items-center justify-between mb-1">
            <span class="font-medium text-sm text-[var(--text)]">#{item.order_number ?? item.id}</span>
            <span class="badge {badgeClass}">{item.status}</span>
          </div>
          <div class="flex items-center justify-between text-xs text-[var(--text-muted)]">
            <span class="tabular-nums">{$fmt.price(item.total)}</span>
            <span>{$fmt.dateTime(item.created_at)}</span>
          </div>
        </div>
      {/each}
      {#if items.length === 0}
        <p class="text-center text-[var(--text-muted)] text-sm py-6">{$t('orders.noOrders')}</p>
      {/if}
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
