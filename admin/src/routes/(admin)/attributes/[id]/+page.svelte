<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { page } from '$app/stores';
  import { t, locale } from 'svelte-i18n';
  import { tr } from '$lib/i18n/entity';
  import {
    attributesApi,
    type Attribute,
    type AttributeOption,
    type AttributeType,
  } from '$lib/api/attributes';
  import { ApiClientError } from '$lib/api/client';
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

  const FIELDS = ['name', 'description'];
  const OPT_FIELDS = ['name'];
  const ATTRIBUTE_TYPES: AttributeType[] = ['text', 'number', 'select', 'multi_select', 'boolean'];

  let id = $derived($page.params.id as string);
  let loading = $state(true);
  let submitting = $state(false);
  let showDeleteAttributeConfirm = $state(false);

  let identifier = $state('');
  let type = $state<AttributeType>('text');
  let unit = $state('');
  let position = $state(0);
  let filterable = $state(false);
  let required = $state(false);
  let translations = $state(emptyTranslations(FIELDS));
  let options = $state<AttributeOption[]>([]);

  // Add option
  let showAddOptionModal = $state(false);
  let optPosition = $state(0);
  let optTranslations = $state(emptyTranslations(OPT_FIELDS));
  let optSubmitting = $state(false);

  // Edit option
  let editOption = $state<AttributeOption | null>(null);
  let editOptPosition = $state(0);
  let editOptTranslations = $state(emptyTranslations(OPT_FIELDS));
  let editOptSubmitting = $state(false);

  // Delete option
  let deleteOptionId = $state<string | null>(null);

  const showUnit = $derived(type === 'number');
  const showOptions = $derived(type === 'select' || type === 'multi_select');

  function optionName(o: AttributeOption): string {
    return tr(o.translations, 'name', $locale) || o.id;
  }

  onMount(async () => {
    try {
      const res = await attributesApi.get(id);
      const a = res.data;
      identifier = a.identifier ?? '';
      type = a.type ?? 'text';
      unit = a.unit ?? '';
      position = a.position;
      filterable = a.filterable ?? false;
      required = a.required ?? false;
      translations = translationsFromArray(a.translations, FIELDS);
      options = a.options ?? [];
    } catch {
      notifications.error($t('attributes.loadOneFailed'));
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
      await attributesApi.update(id, {
        identifier,
        type,
        unit: showUnit ? unit || undefined : undefined,
        position: Number(position),
        filterable,
        required,
        translations: translationsToArray(translations),
      });
      notifications.success($t('attributes.saved'));
    } catch (err: unknown) {
      if (err instanceof ApiClientError && err.status === 409) {
        notifications.error($t('attributes.duplicateIdentifier'));
      } else {
        notifications.error($t('common.saveFailed'));
      }
    } finally {
      submitting = false;
    }
  }

  async function handleDeleteAttribute() {
    try {
      await attributesApi.delete(id);
      notifications.success($t('attributes.deleted'));
      goto(`${base}/attributes`);
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
      const res = await attributesApi.createOption(id, {
        position: Number(optPosition),
        translations: translationsToArray(optTranslations),
      });
      options = [...options, res.data];
      showAddOptionModal = false;
      optPosition = 0;
      optTranslations = emptyTranslations(OPT_FIELDS);
      notifications.success($t('attributes.optionAdded'));
    } catch {
      notifications.error($t('attributes.optionCreateFailed'));
    } finally {
      optSubmitting = false;
    }
  }

  function openEditOption(o: AttributeOption) {
    editOption = o;
    editOptPosition = o.position;
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
      const res = await attributesApi.updateOption(id, editOption.id, {
        position: Number(editOptPosition),
        translations: translationsToArray(editOptTranslations),
      });
      options = options.map((o) => (o.id === editOption!.id ? res.data : o));
      editOption = null;
      notifications.success($t('attributes.optionSaved'));
    } catch {
      notifications.error($t('common.saveFailed'));
    } finally {
      editOptSubmitting = false;
    }
  }

  async function handleDeleteOption() {
    if (!deleteOptionId) return;
    try {
      await attributesApi.deleteOption(id, deleteOptionId);
      options = options.filter((o) => o.id !== deleteOptionId);
      deleteOptionId = null;
      notifications.success($t('attributes.optionDeleted'));
    } catch {
      notifications.error($t('common.deleteFailed'));
    }
  }
</script>

<div class="mb-6">
  <a href="{base}/attributes" class="text-sm text-primary-500 hover:text-primary-400 transition-colors">&larr; {$t('common.back')}</a>
</div>

{#if loading}
  <div class="flex items-center justify-center h-32">
    <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
  </div>
{:else}
  <!-- Attribute form -->
  <div class="card p-6 max-w-lg mb-6">
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-xl font-bold text-[var(--text)]">{$t('attributes.editAttribute')}</h1>
      <button class="btn btn-danger btn-sm" onclick={() => showDeleteAttributeConfirm = true}>{$t('common.delete')}</button>
    </div>

    <form onsubmit={handleSubmit} class="space-y-4">
      <div>
        <label class="label" for="identifier">{$t('attributes.identifier')}</label>
        <input
          id="identifier"
          class="input font-mono"
          type="text"
          bind:value={identifier}
          required
          pattern="^[a-z0-9][a-z0-9_-]*$"
          placeholder="e.g. color"
        />
        <p class="text-xs text-[var(--text-muted)] mt-1">{$t('attributes.identifierHint')}</p>
      </div>

      <div>
        <label class="label" for="type">{$t('attributes.type')}</label>
        <select id="type" class="input" bind:value={type}>
          {#each ATTRIBUTE_TYPES as attrType}
            <option value={attrType}>{$t(`attributes.types.${attrType}`)}</option>
          {/each}
        </select>
      </div>

      {#if showUnit}
        <div>
          <label class="label" for="unit">{$t('attributes.unit')}</label>
          <input
            id="unit"
            class="input"
            type="text"
            bind:value={unit}
            placeholder="e.g. kg, cm, l"
          />
          <p class="text-xs text-[var(--text-muted)] mt-1">{$t('attributes.unitHint')}</p>
        </div>
      {/if}

      <div>
        <label class="label" for="position">{$t('attributes.position')}</label>
        <input id="position" class="input" type="number" min="0" bind:value={position} />
      </div>

      <div class="flex gap-6">
        <label class="flex items-center gap-2 cursor-pointer select-none">
          <input type="checkbox" class="w-4 h-4" bind:checked={filterable} />
          <span class="text-sm text-[var(--text)]">{$t('attributes.filterable')}</span>
        </label>
        <label class="flex items-center gap-2 cursor-pointer select-none">
          <input type="checkbox" class="w-4 h-4" bind:checked={required} />
          <span class="text-sm text-[var(--text)]">{$t('attributes.required')}</span>
        </label>
      </div>

      <div class="border border-[var(--card-border)] rounded-lg p-4">
        <h3 class="text-sm font-semibold text-[var(--text-muted)] mb-3">{$t('common.translations')}</h3>
        <TranslationsInput
          locales={AVAILABLE_LOCALES}
          localeLabels={LOCALE_LABELS}
          primaryLocale={DEFAULT_LOCALE}
          fields={[
            { key: 'name', label: $t('common.name'), type: 'input', required: true },
            { key: 'description', label: $t('common.description'), type: 'textarea' },
          ]}
          bind:value={translations}
        />
      </div>

      <div class="flex gap-3 pt-2">
        <button type="submit" class="btn btn-primary" disabled={submitting}>
          {submitting ? $t('common.saving') : $t('common.save')}
        </button>
        <a href="{base}/attributes" class="btn btn-secondary">{$t('common.cancel')}</a>
      </div>
    </form>
  </div>

  <!-- Options section (only for select / multi_select) -->
  {#if showOptions}
    <div class="card p-6 max-w-lg">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-lg font-semibold text-[var(--text)]">{$t('attributes.options')}</h2>
        <button class="btn btn-secondary btn-sm" onclick={() => showAddOptionModal = true}>
          {$t('attributes.addOption')}
        </button>
      </div>

      {#if options.length === 0}
        <p class="text-sm text-[var(--text-muted)]">{$t('attributes.noOptions')}</p>
      {:else}
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-[var(--card-border)]">
            <thead>
              <tr>
                <th class="table-header">{$t('common.name')}</th>
                <th class="table-header">{$t('attributes.position')}</th>
                <th class="px-4 py-2"></th>
              </tr>
            </thead>
            <tbody class="divide-y divide-[var(--card-border)]">
              {#each options as o}
                <tr>
                  <td class="px-4 py-2 text-sm text-[var(--text-muted)]">{optionName(o)}</td>
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
{/if}

<!-- Delete attribute confirm -->
<ConfirmModal
  open={showDeleteAttributeConfirm}
  title={$t('attributes.deleteTitle')}
  message={$t('attributes.deleteMessage')}
  onConfirm={handleDeleteAttribute}
  onCancel={() => showDeleteAttributeConfirm = false}
/>

<!-- Delete option confirm -->
<ConfirmModal
  open={!!deleteOptionId}
  title={$t('attributes.deleteOptionTitle')}
  message={$t('attributes.deleteOptionMessage')}
  onConfirm={handleDeleteOption}
  onCancel={() => deleteOptionId = null}
/>

<!-- Add option modal -->
<Modal open={showAddOptionModal} title={$t('attributes.addOptionTitle')} onClose={() => showAddOptionModal = false}>
  <form onsubmit={handleAddOption} class="space-y-4">
    <div class="border border-[var(--card-border)] rounded-lg p-4">
      <h3 class="text-sm font-semibold text-[var(--text-muted)] mb-3">{$t('common.translations')}</h3>
      <TranslationsInput
        locales={AVAILABLE_LOCALES}
        localeLabels={LOCALE_LABELS}
        primaryLocale={DEFAULT_LOCALE}
        fields={[{ key: 'name', label: $t('common.name'), type: 'input', required: true }]}
        bind:value={optTranslations}
      />
    </div>
    <div>
      <label class="label" for="opt-position">{$t('attributes.position')}</label>
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

<!-- Edit option modal -->
{#if editOption}
  <Modal open={true} title={$t('attributes.editOption')} onClose={() => editOption = null}>
    <form onsubmit={handleUpdateOption} class="space-y-4">
      <div class="border border-[var(--card-border)] rounded-lg p-4">
        <h3 class="text-sm font-semibold text-[var(--text-muted)] mb-3">{$t('common.translations')}</h3>
        <TranslationsInput
          locales={AVAILABLE_LOCALES}
          localeLabels={LOCALE_LABELS}
          primaryLocale={DEFAULT_LOCALE}
          fields={[{ key: 'name', label: $t('common.name'), type: 'input', required: true }]}
          bind:value={editOptTranslations}
        />
      </div>
      <div>
        <label class="label" for="edit-opt-position">{$t('attributes.position')}</label>
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
