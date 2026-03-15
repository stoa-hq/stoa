<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { page } from '$app/stores';
  import { t, locale } from 'svelte-i18n';
  import { paymentApi } from '$lib/api/payment';
  import { notifications } from '$lib/stores/notifications';
  import ConfirmModal from '$lib/components/ConfirmModal.svelte';
  import TranslationsInput from '$lib/components/TranslationsInput.svelte';
  import PluginSlot from '$lib/components/PluginSlot.svelte';
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
      notifications.error($t('payment.loadOneFailed'));
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
      await paymentApi.update(id, {
        provider: form.provider,
        active: form.active,
        translations: translationsToArray(translations),
      } as any);
      notifications.success($t('payment.saved'));
    } catch (e) {
      notifications.error($t('common.saveFailed'));
    } finally {
      submitting = false;
    }
  }

  async function handleDelete() {
    try {
      await paymentApi.delete(id);
      notifications.success($t('payment.deleted'));
      goto(`${base}/payment`);
    } catch (e) {
      notifications.error($t('common.deleteFailed'));
    }
  }
</script>

<div class="mb-6">
  <a href="{base}/payment" class="text-sm text-primary-500 hover:text-primary-400 transition-colors">&larr; {$t('common.back')}</a>
</div>

{#if loading}
  <div class="flex items-center justify-center h-32">
    <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
  </div>
{:else}
  <div class="card p-6 max-w-2xl">
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-xl font-bold text-[var(--text)]">{$t('payment.editPayment')}</h1>
      <button class="btn btn-danger btn-sm" onclick={() => showDeleteConfirm = true}>{$t('common.delete')}</button>
    </div>

    <form onsubmit={handleSubmit} class="space-y-4">
      <div>
        <label class="label" for="provider">{$t('common.provider')} *</label>
        <input id="provider" class="input" type="text" bind:value={form.provider} required />
      </div>

      <div class="border border-[var(--card-border)] rounded-lg p-4">
        <h3 class="text-sm font-semibold text-[var(--text-muted)] mb-3">{$t('common.translations')}</h3>
        <TranslationsInput
          locales={AVAILABLE_LOCALES}
          localeLabels={LOCALE_LABELS}
          primaryLocale={DEFAULT_LOCALE}
          fields={[
            { key: 'name', label: $t('common.name'), type: 'input', required: true },
            { key: 'description', label: $t('common.description'), type: 'textarea', rows: 3 },
          ]}
          bind:value={translations}
        />
      </div>

      <div class="flex items-center gap-2">
        <input id="active" type="checkbox" bind:checked={form.active} class="h-4 w-4 rounded border-gray-300 text-primary-600" />
        <label for="active" class="text-sm text-[var(--text-muted)]">{$t('common.active')}</label>
      </div>

      <div class="flex gap-3 pt-2">
        <button type="submit" class="btn btn-primary" disabled={submitting}>
          {submitting ? $t('common.saving') : $t('common.save')}
        </button>
        <a href="{base}/payment" class="btn btn-secondary">{$t('common.cancel')}</a>
      </div>
    </form>
  </div>

  <!-- Plugin UI extensions for payment settings -->
  <PluginSlot
    slot="admin:payment:settings"
    context={{ paymentMethodId: id, provider: form.provider }}
  />
{/if}

<ConfirmModal
  open={showDeleteConfirm}
  title={$t('payment.deleteTitle')}
  message={$t('payment.deleteMessage')}
  onConfirm={handleDelete}
  onCancel={() => showDeleteConfirm = false}
/>
