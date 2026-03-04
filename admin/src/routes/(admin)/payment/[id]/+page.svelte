<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { page } from '$app/stores';
  import { paymentApi } from '$lib/api/payment';
  import { notifications } from '$lib/stores/notifications';
  import ConfirmModal from '$lib/components/ConfirmModal.svelte';
  import TranslationsInput from '$lib/components/TranslationsInput.svelte';
  import {
    AVAILABLE_LOCALES,
    DEFAULT_LOCALE,
    LOCALE_LABELS,
    emptyTranslations,
    translationsFromArray,
    translationsToArray,
  } from '$lib/config';

  const FIELDS = ['name', 'description'];

  let id = $derived($page.params.id as string);
  let loading = $state(true);
  let submitting = $state(false);
  let showDeleteConfirm = $state(false);

  let form = $state({
    provider: '',
    active: true,
  });

  let translations = $state(emptyTranslations(FIELDS));

  onMount(async () => {
    try {
      const res = await paymentApi.get(id);
      const method = res.data;
      form = {
        provider: method.provider ?? '',
        active: method.active ?? true,
      };
      translations = translationsFromArray(method.translations, FIELDS);
    } catch (e) {
      notifications.error('Zahlungsmethode konnte nicht geladen werden.');
    } finally {
      loading = false;
    }
  });

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    if (!translations[DEFAULT_LOCALE].name.trim()) {
      notifications.error(`Bitte Name auf ${LOCALE_LABELS[DEFAULT_LOCALE]} ausfüllen.`);
      return;
    }
    submitting = true;
    try {
      await paymentApi.update(id, {
        provider: form.provider,
        active: form.active,
        translations: translationsToArray(translations),
      } as any);
      notifications.success('Zahlungsmethode gespeichert.');
    } catch (e) {
      notifications.error('Speichern fehlgeschlagen.');
    } finally {
      submitting = false;
    }
  }

  async function handleDelete() {
    try {
      await paymentApi.delete(id);
      notifications.success('Zahlungsmethode gelöscht.');
      goto(`${base}/payment`);
    } catch (e) {
      notifications.error('Löschen fehlgeschlagen.');
    }
  }
</script>

<div class="mb-6">
  <a href="{base}/payment" class="text-sm text-primary-600 hover:underline">← Zurück</a>
</div>

{#if loading}
  <div class="flex items-center justify-center h-32">
    <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
  </div>
{:else}
  <div class="card p-6 max-w-2xl">
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-xl font-bold text-gray-900">Zahlungsmethode bearbeiten</h1>
      <button class="btn btn-danger btn-sm" onclick={() => showDeleteConfirm = true}>Löschen</button>
    </div>

    <form onsubmit={handleSubmit} class="space-y-4">
      <div>
        <label class="label" for="provider">Provider *</label>
        <input id="provider" class="input" type="text" bind:value={form.provider} required />
      </div>

      <div class="border border-gray-200 rounded-lg p-4">
        <h3 class="text-sm font-semibold text-gray-700 mb-3">Übersetzungen</h3>
        <TranslationsInput
          locales={AVAILABLE_LOCALES}
          localeLabels={LOCALE_LABELS}
          primaryLocale={DEFAULT_LOCALE}
          fields={[
            { key: 'name', label: 'Name', type: 'input', required: true },
            { key: 'description', label: 'Beschreibung', type: 'textarea', rows: 3 },
          ]}
          bind:value={translations}
        />
      </div>

      <div class="flex items-center gap-2">
        <input id="active" type="checkbox" bind:checked={form.active} class="h-4 w-4 rounded border-gray-300 text-primary-600" />
        <label for="active" class="text-sm text-gray-700">Aktiv</label>
      </div>

      <div class="flex gap-3 pt-2">
        <button type="submit" class="btn btn-primary" disabled={submitting}>
          {submitting ? 'Speichern...' : 'Speichern'}
        </button>
        <a href="{base}/payment" class="btn btn-secondary">Abbrechen</a>
      </div>
    </form>
  </div>
{/if}

<ConfirmModal
  open={showDeleteConfirm}
  title="Zahlungsmethode löschen"
  message="Soll diese Zahlungsmethode wirklich gelöscht werden?"
  onConfirm={handleDelete}
  onCancel={() => showDeleteConfirm = false}
/>
