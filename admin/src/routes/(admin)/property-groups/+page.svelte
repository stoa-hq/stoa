<script lang="ts">
  import { onMount } from 'svelte';
  import { base } from '$app/paths';
  import { t, locale } from 'svelte-i18n';
  import { propertyGroupsApi, type PropertyGroup } from '$lib/api/property-groups';
  import { notifications } from '$lib/stores/notifications';
  import { tr } from '$lib/i18n/entity';
  import SearchBar from '$lib/components/SearchBar.svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';

  let groups = $state<PropertyGroup[]>([]);
  let loading = $state(true);
  let search = $state('');

  const filtered = $derived(
    search
      ? groups.filter(g => {
          const name = tr(g.translations, 'name', $locale) || '';
          return name.toLowerCase().includes(search.toLowerCase());
        })
      : groups
  );

  onMount(async () => {
    try {
      const res = await propertyGroupsApi.list();
      groups = res.data ?? [];
    } catch {
      notifications.error($t('propertyGroups.loadFailed'));
    } finally {
      loading = false;
    }
  });

  function groupName(g: PropertyGroup): string {
    return tr(g.translations, 'name', $locale) || g.id;
  }
</script>

<div class="flex items-center justify-between mb-6">
  <h1 class="text-xl font-bold text-[var(--text)]">{$t('propertyGroups.title')}</h1>
  <a href="{base}/property-groups/new" class="btn btn-primary btn-sm">{$t('propertyGroups.newGroupButton')}</a>
</div>

{#if loading}
  <div class="card p-6 space-y-3">
    {#each Array(3) as _}
      <Skeleton height="h-12" />
    {/each}
  </div>
{:else if groups.length === 0}
  <div class="card p-6 text-center text-[var(--text-muted)]">
    {$t('propertyGroups.noGroups')}
  </div>
{:else}
  <div class="card p-6">
    <div class="mb-4">
      <SearchBar value={search} onSearch={(v) => search = v} debounce={200} />
    </div>
    <div class="overflow-hidden">
      <table class="min-w-full divide-y divide-[var(--card-border)]">
        <thead>
          <tr>
            <th class="table-header">{$t('common.name')}</th>
            <th class="table-header">{$t('propertyGroups.identifier')}</th>
            <th class="table-header">{$t('common.position')}</th>
            <th class="table-header">{$t('propertyGroups.optionCount')}</th>
            <th class="table-header"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-[var(--card-border)]">
          {#each filtered as g}
            <tr class="table-row">
              <td class="table-cell font-medium text-[var(--text)]">{groupName(g)}</td>
              <td class="table-cell">
                <span class="font-mono text-xs bg-[var(--card-border)] text-[var(--text-muted)] px-1.5 py-0.5 rounded">{g.identifier}</span>
              </td>
              <td class="table-cell text-[var(--text-muted)] tabular-nums">{g.position}</td>
              <td class="table-cell text-[var(--text-muted)] tabular-nums">{g.options?.length ?? 0}</td>
              <td class="table-cell text-right">
                <a href="{base}/property-groups/{g.id}" class="text-primary-500 hover:text-primary-400 hover:underline text-sm">{$t('common.edit')}</a>
              </td>
            </tr>
          {/each}
          {#if filtered.length === 0}
            <tr>
              <td colspan="5" class="table-cell text-center text-[var(--text-muted)] py-6">{$t('propertyGroups.noGroups')}</td>
            </tr>
          {/if}
        </tbody>
      </table>
    </div>
  </div>
{/if}
