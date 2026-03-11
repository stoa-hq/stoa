<script lang="ts">
  import { onMount } from 'svelte';
  import { t } from 'svelte-i18n';
  import { auditApi } from '$lib/api/audit';
  import { notifications } from '$lib/stores/notifications';
  import { fmt } from '$lib/i18n/formatters';
  import Pagination from '$lib/components/Pagination.svelte';

  let items = $state<any[]>([]);
  let meta = $state<any>(null);
  let currentPage = $state(1);
  let limit = $state(25);
  let loading = $state(true);

  async function load() {
    loading = true;
    try {
      const res = await auditApi.list({ page: currentPage, limit });
      items = res.data ?? [];
      meta = res.meta ?? null;
    } catch (e) {
      notifications.error($t('audit.loadFailed'));
    } finally {
      loading = false;
    }
  }

  onMount(load);

  function handlePageChange(p: number) {
    currentPage = p;
    load();
  }

  function formatUser(entry: any) {
    const type = entry.user_type ?? 'unknown';
    const id = entry.user_id ? String(entry.user_id).substring(0, 8) : '—';
    return `${type}:${id}`;
  }
</script>

<div class="flex items-center justify-between mb-6">
  <h1 class="text-2xl font-bold text-gray-900">{$t('audit.title')}</h1>
</div>

<div class="card p-6">
  {#if loading}
    <div class="flex items-center justify-center h-32">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
    </div>
  {:else}
    <div class="overflow-x-auto">
      <table class="min-w-full divide-y divide-gray-200">
        <thead>
          <tr>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{$t('audit.timestamp')}</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{$t('audit.user')}</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{$t('audit.action')}</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{$t('audit.entity')}</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{$t('audit.entityId')}</th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          {#each items as entry}
            <tr class="hover:bg-gray-50">
              <td class="px-4 py-3 text-sm text-gray-500 whitespace-nowrap">{$fmt.dateTime(entry.created_at)}</td>
              <td class="px-4 py-3 text-sm font-mono text-gray-700">{formatUser(entry)}</td>
              <td class="px-4 py-3 text-sm">
                <span class="badge badge-blue">{entry.action}</span>
              </td>
              <td class="px-4 py-3 text-sm text-gray-700">{entry.entity_type ?? '—'}</td>
              <td class="px-4 py-3 text-sm font-mono text-gray-500">
                {entry.entity_id ? String(entry.entity_id).substring(0, 12) + '...' : '—'}
              </td>
            </tr>
          {/each}
          {#if items.length === 0}
            <tr>
              <td colspan="5" class="px-4 py-6 text-center text-gray-400 text-sm">{$t('audit.noEntries')}</td>
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
