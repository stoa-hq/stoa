<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { page } from '$app/stores';
  import { t, locale } from 'svelte-i18n';
  import { tr } from '$lib/i18n/entity';
  import {
    propertyGroupsApi,
    type PropertyGroup,
    type PropertyOption,
  } from '$lib/api/property-groups';
  import { notifications } from '$lib/stores/notifications';
  import TranslationsInput from '$lib/components/TranslationsInput.svelte';
  import ConfirmModal from '$lib/components/ConfirmModal.svelte';
  import Modal from '$lib/components/Modal.svelte';
  import {
    AVAILABLE_LOCALES,
    DEFAULT_LOCALE,
    LOCALE_LABELS,
    emptyTranslations,
    translationsFromArray,
    translationsToArray,
  } from '$lib/config';

  const FIELDS = ['name'];
  const OPT_FIELDS = ['name'];

  let id = $derived($page.params.id as string);
  let loading = $state(true);
  let submitting = $state(false);
  let showDeleteGroupConfirm = $state(false);

  let position = $state(0);
  let translations = $state(emptyTranslations(FIELDS));
  let options = $state<PropertyOption[]>([]);

  // Neue Ausprägung
  let showAddOptionModal = $state(false);
  let optPosition = $state(0);
  let optColorHex = $state('');
  let optTranslations = $state(emptyTranslations(OPT_FIELDS));
  let optSubmitting = $state(false);

  // Ausprägung bearbeiten
  let editOption = $state<PropertyOption | null>(null);
  let editOptPosition = $state(0);
  let editOptColorHex = $state('');
  let editOptTranslations = $state(emptyTranslations(OPT_FIELDS));
  let editOptSubmitting = $state(false);

  // Ausprägung löschen
  let deleteOptionId = $state<string | null>(null);

  function optionName(o: PropertyOption): string {
    return tr(o.translations, 'name', $locale) || o.id;
  }

  onMount(async () => {
    try {
      const res = await propertyGroupsApi.get(id);
      const g = res.data;
      position = g.position;
      translations = translationsFromArray(g.translations, FIELDS);
      options = g.options ?? [];
    } catch {
      notifications.error($t('propertyGroups.loadOneFailed'));
    } finally {
      loading = false;
    }
  });

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    if (!translations[DEFAULT_LOCALE].name.trim()) {
      notifications.error($t('common.pleaseNameGerman'));
      return;
    }
    submitting = true;
    try {
      await propertyGroupsApi.update(id, {
        position: Number(position),
        translations: translationsToArray(translations),
      });
      notifications.success($t('propertyGroups.saved'));
    } catch {
      notifications.error($t('common.saveFailed'));
    } finally {
      submitting = false;
    }
  }

  async function handleDeleteGroup() {
    try {
      await propertyGroupsApi.delete(id);
      notifications.success($t('propertyGroups.deleted'));
      goto(`${base}/property-groups`);
    } catch {
      notifications.error($t('common.deleteFailed'));
    }
  }

  async function handleAddOption(e: SubmitEvent) {
    e.preventDefault();
    if (!optTranslations[DEFAULT_LOCALE].name.trim()) {
      notifications.error($t('common.pleaseNameGerman'));
      return;
    }
    optSubmitting = true;
    try {
      const res = await propertyGroupsApi.createOption(id, {
        position: Number(optPosition),
        color_hex: optColorHex || undefined,
        translations: translationsToArray(optTranslations),
      });
      options = [...options, res.data];
      showAddOptionModal = false;
      optPosition = 0;
      optColorHex = '';
      optTranslations = emptyTranslations(OPT_FIELDS);
      notifications.success($t('propertyGroups.optionAdded'));
    } catch {
      notifications.error($t('propertyGroups.optionCreateFailed'));
    } finally {
      optSubmitting = false;
    }
  }

  function openEditOption(o: PropertyOption) {
    editOption = o;
    editOptPosition = o.position;
    editOptColorHex = o.color_hex ?? '';
    editOptTranslations = translationsFromArray(o.translations, OPT_FIELDS);
  }

  async function handleUpdateOption(e: SubmitEvent) {
    e.preventDefault();
    if (!editOption) return;
    if (!editOptTranslations[DEFAULT_LOCALE].name.trim()) {
      notifications.error($t('common.pleaseNameGerman'));
      return;
    }
    editOptSubmitting = true;
    try {
      const res = await propertyGroupsApi.updateOption(id, editOption.id, {
        position: Number(editOptPosition),
        color_hex: editOptColorHex || undefined,
        translations: translationsToArray(editOptTranslations),
      });
      options = options.map((o) => (o.id === editOption!.id ? res.data : o));
      editOption = null;
      notifications.success($t('propertyGroups.optionSaved'));
    } catch {
      notifications.error($t('common.saveFailed'));
    } finally {
      editOptSubmitting = false;
    }
  }

  async function handleDeleteOption() {
    if (!deleteOptionId) return;
    try {
      await propertyGroupsApi.deleteOption(id, deleteOptionId);
      options = options.filter((o) => o.id !== deleteOptionId);
      deleteOptionId = null;
      notifications.success($t('propertyGroups.optionDeleted'));
    } catch {
      notifications.error($t('common.deleteFailed'));
    }
  }
</script>

<div class="mb-6">
  <a href="{base}/property-groups" class="text-sm text-primary-500 hover:text-primary-400 transition-colors">&larr; {$t('common.back')}</a>
</div>

{#if loading}
  <div class="flex items-center justify-center h-32">
    <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
  </div>
{:else}
  <!-- Gruppe bearbeiten -->
  <div class="card p-6 max-w-lg mb-6">
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-xl font-bold text-[var(--text)]">{$t('propertyGroups.editGroup')}</h1>
      <button class="btn btn-danger btn-sm" onclick={() => showDeleteGroupConfirm = true}>{$t('common.delete')}</button>
    </div>

    <form onsubmit={handleSubmit} class="space-y-4">
      <div>
        <label class="label" for="position">{$t('common.position')}</label>
        <input id="position" class="input" type="number" min="0" bind:value={position} />
      </div>

      <div class="border border-[var(--card-border)] rounded-lg p-4">
        <h3 class="text-sm font-semibold text-[var(--text-muted)] mb-3">{$t('propertyGroups.nameTranslations')}</h3>
        <TranslationsInput
          locales={AVAILABLE_LOCALES}
          localeLabels={LOCALE_LABELS}
          primaryLocale={DEFAULT_LOCALE}
          fields={[{ key: 'name', label: $t('common.name'), type: 'input', required: true }]}
          bind:value={translations}
        />
      </div>

      <div class="flex gap-3 pt-2">
        <button type="submit" class="btn btn-primary" disabled={submitting}>
          {submitting ? $t('common.saving') : $t('common.save')}
        </button>
        <a href="{base}/property-groups" class="btn btn-secondary">{$t('common.cancel')}</a>
      </div>
    </form>
  </div>

  <!-- Ausprägungen -->
  <div class="card p-6 max-w-lg">
    <div class="flex items-center justify-between mb-4">
      <h2 class="text-lg font-semibold text-[var(--text)]">{$t('propertyGroups.options')}</h2>
      <button class="btn btn-secondary btn-sm" onclick={() => showAddOptionModal = true}>
        {$t('propertyGroups.addOption')}
      </button>
    </div>

    {#if options.length === 0}
      <p class="text-sm text-[var(--text-muted)]">{$t('propertyGroups.noOptions')}</p>
    {:else}
      <div class="overflow-x-auto">
        <table class="min-w-full divide-y divide-[var(--card-border)]">
          <thead>
            <tr>
              <th class="table-header">{$t('common.name')}</th>
              <th class="table-header">{$t('propertyGroups.color')}</th>
              <th class="table-header">{$t('common.position')}</th>
              <th class="px-4 py-2"></th>
            </tr>
          </thead>
          <tbody class="divide-y divide-[var(--card-border)]">
            {#each options as o}
              <tr>
                <td class="px-4 py-2 text-sm text-[var(--text-muted)]">{optionName(o)}</td>
                <td class="px-4 py-2 text-sm">
                  {#if o.color_hex}
                    <span class="inline-flex items-center gap-2">
                      <span class="w-4 h-4 rounded-full border border-gray-300 inline-block" style="background:{o.color_hex}"></span>
                      {o.color_hex}
                    </span>
                  {:else}
                    <span class="text-[var(--text-muted)]">—</span>
                  {/if}
                </td>
                <td class="px-4 py-2 text-sm text-[var(--text-muted)]">{o.position}</td>
                <td class="px-4 py-2 text-right flex gap-2 justify-end">
                  <button
                    class="text-primary-500 hover:text-primary-400 transition-colors text-xs"
                    onclick={() => openEditOption(o)}
                  >{$t('common.edit')}</button>
                  <button
                    class="text-red-600 hover:underline text-xs"
                    onclick={() => deleteOptionId = o.id}
                  >{$t('common.delete')}</button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
{/if}

<!-- Gruppe löschen -->
<ConfirmModal
  open={showDeleteGroupConfirm}
  title={$t('propertyGroups.deleteTitle')}
  message={$t('propertyGroups.deleteMessage')}
  onConfirm={handleDeleteGroup}
  onCancel={() => showDeleteGroupConfirm = false}
/>

<!-- Ausprägung löschen -->
<ConfirmModal
  open={!!deleteOptionId}
  title={$t('propertyGroups.deleteOptionTitle')}
  message={$t('propertyGroups.deleteOptionMessage')}
  onConfirm={handleDeleteOption}
  onCancel={() => deleteOptionId = null}
/>

<!-- Ausprägung hinzufügen -->
<Modal open={showAddOptionModal} title={$t('propertyGroups.addOptionTitle')} onClose={() => showAddOptionModal = false}>
  <form onsubmit={handleAddOption} class="space-y-4">
    <div class="border border-[var(--card-border)] rounded-lg p-4">
      <h3 class="text-sm font-semibold text-[var(--text-muted)] mb-3">{$t('propertyGroups.nameTranslations')}</h3>
      <TranslationsInput
        locales={AVAILABLE_LOCALES}
        localeLabels={LOCALE_LABELS}
        primaryLocale={DEFAULT_LOCALE}
        fields={[{ key: 'name', label: $t('common.name'), type: 'input', required: true }]}
        bind:value={optTranslations}
      />
    </div>
    <div>
      <label class="label" for="opt-color">{$t('propertyGroups.colorHex')}</label>
      <div class="flex gap-2 items-center">
        <input id="opt-color-picker" type="color" bind:value={optColorHex} class="h-9 w-12 cursor-pointer rounded border border-gray-300" />
        <input id="opt-color" class="input flex-1" type="text" bind:value={optColorHex} placeholder="#FF0000" />
      </div>
    </div>
    <div>
      <label class="label" for="opt-position">{$t('common.position')}</label>
      <input id="opt-position" class="input" type="number" min="0" bind:value={optPosition} />
    </div>
    <div class="flex gap-3 pt-2">
      <button type="submit" class="btn btn-primary" disabled={optSubmitting}>
        {optSubmitting ? $t('common.saving') : $t('common.add')}
      </button>
      <button type="button" class="btn btn-secondary" onclick={() => showAddOptionModal = false}>{$t('common.cancel')}</button>
    </div>
  </form>
</Modal>

<!-- Ausprägung bearbeiten -->
{#if editOption}
  <Modal open={true} title={$t('propertyGroups.editOption')} onClose={() => editOption = null}>
    <form onsubmit={handleUpdateOption} class="space-y-4">
      <div class="border border-[var(--card-border)] rounded-lg p-4">
        <h3 class="text-sm font-semibold text-[var(--text-muted)] mb-3">{$t('propertyGroups.nameTranslations')}</h3>
        <TranslationsInput
          locales={AVAILABLE_LOCALES}
          localeLabels={LOCALE_LABELS}
          primaryLocale={DEFAULT_LOCALE}
          fields={[{ key: 'name', label: $t('common.name'), type: 'input', required: true }]}
          bind:value={editOptTranslations}
        />
      </div>
      <div>
        <label class="label" for="edit-opt-color">{$t('propertyGroups.colorHex')}</label>
        <div class="flex gap-2 items-center">
          <input id="edit-opt-color-picker" type="color" bind:value={editOptColorHex} class="h-9 w-12 cursor-pointer rounded border border-gray-300" />
          <input id="edit-opt-color" class="input flex-1" type="text" bind:value={editOptColorHex} placeholder="#FF0000" />
        </div>
      </div>
      <div>
        <label class="label" for="edit-opt-position">{$t('common.position')}</label>
        <input id="edit-opt-position" class="input" type="number" min="0" bind:value={editOptPosition} />
      </div>
      <div class="flex gap-3 pt-2">
        <button type="submit" class="btn btn-primary" disabled={editOptSubmitting}>
          {editOptSubmitting ? $t('common.saving') : $t('common.save')}
        </button>
        <button type="button" class="btn btn-secondary" onclick={() => editOption = null}>{$t('common.cancel')}</button>
      </div>
    </form>
  </Modal>
{/if}
