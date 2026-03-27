<script lang="ts">
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { t } from 'svelte-i18n';
  import { attributesApi, type AttributeType } from '$lib/api/attributes';
  import { ApiClientError } from '$lib/api/client';
  import { notifications } from '$lib/stores/notifications';
  import TranslationsInput from '$lib/components/TranslationsInput.svelte';
  import {
    AVAILABLE_LOCALES,
    DEFAULT_LOCALE,
    LOCALE_LABELS,
    emptyTranslations,
    translationsToArray,
  } from '$lib/config';

  const FIELDS = ['name', 'description'];
  const ATTRIBUTE_TYPES: AttributeType[] = ['text', 'number', 'select', 'multi_select', 'boolean'];

  let identifier = $state('');
  let type = $state<AttributeType>('text');
  let unit = $state('');
  let position = $state(0);
  let filterable = $state(false);
  let required = $state(false);
  let translations = $state(emptyTranslations(FIELDS));
  let submitting = $state(false);

  const showUnit = $derived(type === 'number');

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    if (!translations[DEFAULT_LOCALE].name.trim()) {
      notifications.error($t('common.pleaseNameGerman'));
      return;
    }
    submitting = true;
    try {
      const res = await attributesApi.create({
        identifier,
        type,
        unit: showUnit ? unit || undefined : undefined,
        position: Number(position),
        filterable,
        required,
        translations: translationsToArray(translations),
      });
      notifications.success($t('attributes.created'));
      goto(`${base}/attributes/${res.data.id}`);
    } catch (err: unknown) {
      if (err instanceof ApiClientError && err.status === 409) {
        notifications.error($t('attributes.duplicateIdentifier'));
      } else {
        notifications.error($t('common.createFailed'));
      }
    } finally {
      submitting = false;
    }
  }
</script>

<div class="mb-6">
  <a href="{base}/attributes" class="text-sm text-primary-500 hover:text-primary-400 transition-colors">&larr; {$t('common.back')}</a>
</div>

<div class="card p-6 max-w-lg">
  <h1 class="text-xl font-bold text-[var(--text)] mb-6">{$t('attributes.newAttribute')}</h1>

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
        {submitting ? $t('common.creating') : $t('common.create')}
      </button>
      <a href="{base}/attributes" class="btn btn-secondary">{$t('common.cancel')}</a>
    </div>
  </form>
</div>
