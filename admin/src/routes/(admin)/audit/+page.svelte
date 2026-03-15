<script lang="ts">
  import { onMount } from 'svelte';
  import { t } from 'svelte-i18n';
  import { auditApi } from '$lib/api/audit';
  import { notifications } from '$lib/stores/notifications';
  import { fmt } from '$lib/i18n/formatters';
  import Pagination from '$lib/components/Pagination.svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';

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
  <h1 class="text-2xl font-bold text-[var(--text)]">{$t('audit.title')}</h1>
</div>

<div class="card p-6">
  {#if loading}
    <div class="space-y-3">
      {#each Array(5) as _}
        <Skeleton height="h-10" />
      {/each}
    </div>
  {:else}
    <div class="overflow-x-auto">
      <table class="min-w-full divide-y divide-[var(--card-border)]">
        <thead>
          <tr>
            <th class="table-header">{$t('audit.timestamp')}</th>
            <th class="table-header">{$t('audit.user')}</th>
            <th class="table-header">{$t('audit.action')}</th>
            <th class="table-header">{$t('audit.entity')}</th>
            <th class="table-header">{$t('audit.entityId')}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-[var(--card-border)]">
          {#each items as entry}
            <tr class="table-row">
              <td class="table-cell text-[var(--text-muted)] whitespace-nowrap">{$fmt.dateTime(entry.created_at)}</td>
              <td class="table-cell font-mono text-xs text-[var(--text-muted)]">{formatUser(entry)}</td>
              <td class="table-cell">
                <span class="badge badge-blue">{entry.action}</span>
              </td>
              <td class="table-cell text-[var(--text-muted)]">{entry.entity_type ?? '—'}</td>
              <td class="table-cell font-mono text-xs text-[var(--text-muted)]">
                {entry.entity_id ? String(entry.entity_id).substring(0, 12) + '...' : '—'}
              </td>
            </tr>
          {/each}
          {#if items.length === 0}
            <tr>
              <td colspan="5" class="table-cell text-center text-[var(--text-muted)] py-6">{$t('audit.noEntries')}</td>
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
