<script lang="ts">
  import { onMount } from 'svelte';
  import { base } from '$app/paths';
  import { t, locale } from 'svelte-i18n';
  import { propertyGroupsApi, type PropertyGroup } from '$lib/api/property-groups';
  import { notifications } from '$lib/stores/notifications';
  import { tr } from '$lib/i18n/entity';

  let groups = $state<PropertyGroup[]>([]);
  let loading = $state(true);

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
  <h1 class="text-xl font-bold text-gray-900">{$t('propertyGroups.title')}</h1>
  <a href="{base}/property-groups/new" class="btn btn-primary btn-sm">{$t('propertyGroups.newGroupButton')}</a>
</div>

{#if loading}
  <div class="flex items-center justify-center h-32">
    <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
  </div>
{:else if groups.length === 0}
  <div class="card p-6 text-center text-gray-400">
    {$t('propertyGroups.noGroups')}
  </div>
{:else}
  <div class="card overflow-hidden">
    <table class="min-w-full divide-y divide-gray-200">
      <thead class="bg-gray-50">
        <tr>
          <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{$t('common.name')}</th>
          <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{$t('common.position')}</th>
          <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{$t('propertyGroups.optionCount')}</th>
          <th class="px-6 py-3"></th>
        </tr>
      </thead>
      <tbody class="divide-y divide-gray-200">
        {#each groups as g}
          <tr class="hover:bg-gray-50">
            <td class="px-6 py-3 text-sm font-medium text-gray-900">{groupName(g)}</td>
            <td class="px-6 py-3 text-sm text-gray-500">{g.position}</td>
            <td class="px-6 py-3 text-sm text-gray-500">{g.options?.length ?? 0}</td>
            <td class="px-6 py-3 text-right">
              <a href="{base}/property-groups/{g.id}" class="text-primary-600 hover:underline text-sm">{$t('common.edit')}</a>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
{/if}
