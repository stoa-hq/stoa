<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { categoriesApi } from '$lib/api/categories';
  import { notifications } from '$lib/stores/notifications';
  import TranslationsInput from '$lib/components/TranslationsInput.svelte';
  import {
    AVAILABLE_LOCALES,
    DEFAULT_LOCALE,
    LOCALE_LABELS,
    emptyTranslations,
    translationsToArray,
  } from '$lib/config';

  const FIELDS = ['name', 'slug', 'description'];

  let allCategories = $state<any[]>([]);
  let submitting = $state(false);

  let form = $state({
    parent_id: '',
    position: 0,
    active: true,
  });

  let translations = $state(emptyTranslations(FIELDS));

  onMount(async () => {
    try {
      const res = await categoriesApi.list({ limit: 200 });
      allCategories = res.data ?? [];
    } catch (e) {
      // ignore
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
      await categoriesApi.create({
        parent_id: form.parent_id || null,
        position: Number(form.position),
        active: form.active,
        translations: translationsToArray(translations),
      });
      notifications.success('Kategorie erstellt.');
      goto(`${base}/categories`);
    } catch (e) {
      notifications.error('Erstellen fehlgeschlagen.');
    } finally {
      submitting = false;
    }
  }
</script>

<div class="mb-6">
  <a href="{base}/categories" class="text-sm text-primary-600 hover:underline">← Zurück</a>
</div>

<div class="card p-6 max-w-2xl">
  <h1 class="text-xl font-bold text-gray-900 mb-6">Neue Kategorie</h1>

  <form onsubmit={handleSubmit} class="space-y-4">
    <div class="border border-gray-200 rounded-lg p-4">
      <h3 class="text-sm font-semibold text-gray-700 mb-3">Übersetzungen</h3>
      <TranslationsInput
        locales={AVAILABLE_LOCALES}
        localeLabels={LOCALE_LABELS}
        primaryLocale={DEFAULT_LOCALE}
        fields={[
          { key: 'name', label: 'Name', type: 'input', required: true },
          { key: 'slug', label: 'Slug', type: 'input', required: true },
          { key: 'description', label: 'Beschreibung', type: 'textarea', rows: 3 },
        ]}
        bind:value={translations}
      />
    </div>

    <div>
      <label class="label" for="parent_id">Elternkategorie</label>
      <select id="parent_id" class="input" bind:value={form.parent_id}>
        <option value="">— Keine —</option>
        {#each allCategories as cat}
          <option value={cat.id}>{cat.translations?.[0]?.name ?? cat.id}</option>
        {/each}
      </select>
    </div>

    <div>
      <label class="label" for="position">Position</label>
      <input id="position" class="input" type="number" min="0" bind:value={form.position} />
    </div>

    <div class="flex items-center gap-2">
      <input id="active" type="checkbox" bind:checked={form.active} class="h-4 w-4 rounded border-gray-300 text-primary-600" />
      <label for="active" class="text-sm text-gray-700">Aktiv</label>
    </div>

    <div class="flex gap-3 pt-2">
      <button type="submit" class="btn btn-primary" disabled={submitting}>
        {submitting ? 'Speichern...' : 'Speichern'}
      </button>
      <a href="{base}/categories" class="btn btn-secondary">Abbrechen</a>
    </div>
  </form>
</div>
