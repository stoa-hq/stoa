<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { page } from '$app/stores';
  import { t, locale } from 'svelte-i18n';
  import { categoriesApi } from '$lib/api/categories';
  import { tr } from '$lib/i18n/entity';
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

  const FIELDS = ['name', 'slug', 'description'];

  let id = $derived($page.params.id as string);
  let loading = $state(true);
  let submitting = $state(false);
  let showDeleteConfirm = $state(false);
  let allCategories = $state<any[]>([]);

  let form = $state({
    parent_id: '',
    position: 0,
    active: true,
  });

  let translations = $state(emptyTranslations(FIELDS));

  onMount(async () => {
    try {
      const [catRes, allRes] = await Promise.all([
        categoriesApi.get(id),
        categoriesApi.list({ limit: 200 }),
      ]);
      const cat = catRes.data;
      allCategories = (allRes.data ?? []).filter((c: any) => c.id !== id);
      form = {
        parent_id: cat.parent_id ?? '',
        position: cat.position ?? 0,
        active: cat.active ?? true,
      };
      translations = translationsFromArray(cat.translations, FIELDS);
    } catch (e) {
      notifications.error($t('categories.loadOneFailed'));
    } finally {
      loading = false;
    }
  });

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    if (!translations[DEFAULT_LOCALE].name.trim()) {
      notifications.error($t('common.pleaseNameInLocale', { values: { locale: LOCALE_LABELS[DEFAULT_LOCALE] } }));
      return;
    }
    submitting = true;
    try {
      await categoriesApi.update(id, {
        parent_id: form.parent_id || null,
        position: Number(form.position),
        active: form.active,
        translations: translationsToArray(translations),
      });
      notifications.success($t('categories.saved'));
    } catch (e) {
      notifications.error($t('common.saveFailed'));
    } finally {
      submitting = false;
    }
  }

  async function handleDelete() {
    try {
      await categoriesApi.delete(id);
      notifications.success($t('categories.deleted'));
      goto(`${base}/categories`);
    } catch (e) {
      notifications.error($t('common.deleteFailed'));
    }
  }
</script>

<div class="mb-6">
  <a href="{base}/categories" class="text-sm text-primary-500 hover:text-primary-400 transition-colors">&larr; {$t('common.back')}</a>
</div>

{#if loading}
  <div class="flex items-center justify-center h-32">
    <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
  </div>
{:else}
  <div class="card p-6 max-w-2xl">
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-xl font-bold text-[var(--text)]">{$t('categories.editCategory')}</h1>
      <button class="btn btn-danger btn-sm" onclick={() => showDeleteConfirm = true}>{$t('common.delete')}</button>
    </div>

    <form onsubmit={handleSubmit} class="space-y-4">
      <div class="border border-[var(--card-border)] rounded-lg p-4">
        <h3 class="text-sm font-semibold text-[var(--text-muted)] mb-3">{$t('common.translations')}</h3>
        <TranslationsInput
          locales={AVAILABLE_LOCALES}
          localeLabels={LOCALE_LABELS}
          primaryLocale={DEFAULT_LOCALE}
          fields={[
            { key: 'name', label: $t('common.name'), type: 'input', required: true },
            { key: 'slug', label: $t('common.slug'), type: 'input', required: true },
            { key: 'description', label: $t('common.description'), type: 'textarea', rows: 3 },
          ]}
          bind:value={translations}
        />
      </div>

      <div>
        <label class="label" for="parent_id">{$t('categories.parentCategory')}</label>
        <select id="parent_id" class="input" bind:value={form.parent_id}>
          <option value="">{$t('common.noSelection')}</option>
          {#each allCategories as cat}
            <option value={cat.id}>{tr(cat.translations, 'name', $locale) || cat.id}</option>
          {/each}
        </select>
      </div>

      <div>
        <label class="label" for="position">{$t('common.position')}</label>
        <input id="position" class="input" type="number" min="0" bind:value={form.position} />
      </div>

      <div class="flex items-center gap-2">
        <input id="active" type="checkbox" bind:checked={form.active} class="h-4 w-4 rounded border-gray-300 text-primary-600" />
        <label for="active" class="text-sm text-[var(--text-muted)]">{$t('common.active')}</label>
      </div>

      <div class="flex gap-3 pt-2">
        <button type="submit" class="btn btn-primary" disabled={submitting}>
          {submitting ? $t('common.saving') : $t('common.save')}
        </button>
        <a href="{base}/categories" class="btn btn-secondary">{$t('common.cancel')}</a>
      </div>
    </form>
  </div>
{/if}

<ConfirmModal
  open={showDeleteConfirm}
  title={$t('categories.deleteTitle')}
  message={$t('categories.deleteMessage')}
  onConfirm={handleDelete}
  onCancel={() => showDeleteConfirm = false}
/>
