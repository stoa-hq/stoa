<script lang="ts">
  import { onMount } from 'svelte';
  import { base } from '$app/paths';
  import { t, locale } from 'svelte-i18n';
  import { attributesApi, type Attribute } from '$lib/api/attributes';
  import { notifications } from '$lib/stores/notifications';
  import { tr } from '$lib/i18n/entity';
  import SearchBar from '$lib/components/SearchBar.svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';

  let attributes = $state<Attribute[]>([]);
  let loading = $state(true);
  let search = $state('');

  const filtered = $derived(
    search
      ? attributes.filter((a) => {
          const name = tr(a.translations, 'name', $locale) || '';
          return (
            name.toLowerCase().includes(search.toLowerCase()) ||
            a.identifier.toLowerCase().includes(search.toLowerCase())
          );
        })
      : attributes
  );

  onMount(async () => {
    try {
      const res = await attributesApi.list();
      attributes = (res.data ?? []).sort((a, b) => a.position - b.position);
    } catch {
      notifications.error($t('attributes.loadFailed'));
    } finally {
      loading = false;
    }
  });

  function attrName(a: Attribute): string {
    return tr(a.translations, 'name', $locale) || a.identifier;
  }

  const TYPE_BADGE_CLASSES: Record<string, string> = {
    text: 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300',
    number: 'bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-300',
    select: 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300',
    multi_select: 'bg-teal-100 text-teal-700 dark:bg-teal-900/40 dark:text-teal-300',
    boolean: 'bg-orange-100 text-orange-700 dark:bg-orange-900/40 dark:text-orange-300',
  };
</script>

<div class="flex items-center justify-between mb-6">
  <h1 class="text-xl font-bold text-[var(--text)]">{$t('attributes.title')}</h1>
  <a href="{base}/attributes/new" class="btn btn-primary btn-sm">{$t('attributes.newAttributeButton')}</a>
</div>

{#if loading}
  <div class="card p-6 space-y-3">
    {#each Array(4) as _}
      <Skeleton height="h-12" />
    {/each}
  </div>
{:else if attributes.length === 0}
  <div class="card p-6 text-center text-[var(--text-muted)]">
    {$t('attributes.noAttributes')}
  </div>
{:else}
  <div class="card p-6">
    <div class="mb-4">
      <SearchBar value={search} onSearch={(v) => search = v} debounce={200} />
    </div>
    <div class="overflow-x-auto">
      <table class="min-w-full divide-y divide-[var(--card-border)]">
        <thead>
          <tr>
            <th class="table-header">{$t('common.name')}</th>
            <th class="table-header">{$t('attributes.identifier')}</th>
            <th class="table-header">{$t('attributes.type')}</th>
            <th class="table-header">{$t('attributes.unit')}</th>
            <th class="table-header">{$t('attributes.filterable')}</th>
            <th class="table-header">{$t('attributes.options')}</th>
            <th class="table-header"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-[var(--card-border)]">
          {#each filtered as a}
            <tr class="table-row">
              <td class="table-cell font-medium text-[var(--text)]">{attrName(a)}</td>
              <td class="table-cell">
                <span class="font-mono text-xs bg-[var(--card-border)] text-[var(--text-muted)] px-1.5 py-0.5 rounded">{a.identifier}</span>
              </td>
              <td class="table-cell">
                <span class="inline-block px-2 py-0.5 rounded text-xs font-medium {TYPE_BADGE_CLASSES[a.type] ?? ''}">
                  {$t(`attributes.types.${a.type}`)}
                </span>
              </td>
              <td class="table-cell text-[var(--text-muted)]">
                {a.unit || '—'}
              </td>
              <td class="table-cell">
                {#if a.filterable}
                  <span class="text-green-600 dark:text-green-400" title={$t('attributes.filterable')}>✓</span>
                {:else}
                  <span class="text-[var(--text-muted)]">—</span>
                {/if}
              </td>
              <td class="table-cell text-[var(--text-muted)] tabular-nums">
                {#if a.type === 'select' || a.type === 'multi_select'}
                  {a.options?.length ?? 0}
                {:else}
                  —
                {/if}
              </td>
              <td class="table-cell text-right">
                <a href="{base}/attributes/{a.id}" class="text-primary-500 hover:text-primary-400 hover:underline text-sm">{$t('common.edit')}</a>
              </td>
            </tr>
          {/each}
          {#if filtered.length === 0}
            <tr>
              <td colspan="7" class="table-cell text-center text-[var(--text-muted)] py-6">{$t('attributes.noAttributes')}</td>
            </tr>
          {/if}
        </tbody>
      </table>
    </div>
  </div>
{/if}
