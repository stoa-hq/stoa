<script lang="ts">
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { t } from 'svelte-i18n';
  import { propertyGroupsApi } from '$lib/api/property-groups';
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

  const FIELDS = ['name'];

  let identifier = $state('');
  let position = $state(0);
  let translations = $state(emptyTranslations(FIELDS));
  let submitting = $state(false);

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    if (!translations[DEFAULT_LOCALE].name.trim()) {
      notifications.error($t('common.pleaseNameGerman'));
      return;
    }
    submitting = true;
    try {
      await propertyGroupsApi.create({
        identifier,
        position: Number(position),
        translations: translationsToArray(translations),
      });
      notifications.success($t('propertyGroups.created'));
      goto(`${base}/property-groups`);
    } catch (err: unknown) {
      if (err instanceof ApiClientError && err.status === 409) {
        notifications.error($t('propertyGroups.duplicateIdentifier'));
      } else {
        notifications.error($t('common.createFailed'));
      }
    } finally {
      submitting = false;
    }
  }
</script>

<div class="mb-6">
  <a href="{base}/property-groups" class="text-sm text-primary-500 hover:text-primary-400 transition-colors">&larr; {$t('common.back')}</a>
</div>

<div class="card p-6 max-w-lg">
  <h1 class="text-xl font-bold text-[var(--text)] mb-6">{$t('propertyGroups.newGroup')}</h1>

  <form onsubmit={handleSubmit} class="space-y-4">
    <div>
      <label class="label" for="identifier">{$t('propertyGroups.identifier')}</label>
      <input
        id="identifier"
        class="input font-mono"
        type="text"
        bind:value={identifier}
        required
        pattern="^[a-z0-9][a-z0-9_-]*$"
        placeholder="e.g. color-group"
      />
      <p class="text-xs text-[var(--text-muted)] mt-1">{$t('propertyGroups.identifierHint')}</p>
    </div>

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
        {submitting ? $t('common.saving') : $t('common.create')}
      </button>
      <a href="{base}/property-groups" class="btn btn-secondary">{$t('common.cancel')}</a>
    </div>
  </form>
</div>
