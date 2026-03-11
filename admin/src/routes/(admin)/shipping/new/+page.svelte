<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { t } from 'svelte-i18n';
  import { shippingApi } from '$lib/api/shipping';
  import { taxApi } from '$lib/api/tax';
  import { notifications } from '$lib/stores/notifications';
  import { fmt } from '$lib/i18n/formatters';
  import TranslationsInput from '$lib/components/TranslationsInput.svelte';
  import {
    AVAILABLE_LOCALES,
    DEFAULT_LOCALE,
    LOCALE_LABELS,
    emptyTranslations,
    translationsToArray,
  } from '$lib/config';

  const FIELDS = ['name', 'description'];

  let submitting = $state(false);
  let form = $state({
    active: true,
  });

  let translations = $state(emptyTranslations(FIELDS));

  let allTaxRules = $state<any[]>([]);
  let selectedTaxRuleId = $state('');
  let priceMode = $state<'gross' | 'net'>('gross');
  let enteredPrice = $state(0);

  const selectedTaxRule = $derived(allTaxRules.find(t => t.id === selectedTaxRuleId));
  const calculatedPrice = $derived((() => {
    if (!selectedTaxRule || !enteredPrice) return null;
    const rate = selectedTaxRule.rate;
    return priceMode === 'gross'
      ? Math.round(enteredPrice * 10000 / (10000 + rate))
      : Math.round(enteredPrice * (10000 + rate) / 10000);
  })());

  onMount(async () => {
    try {
      const taxRes = await taxApi.list({ limit: 200 });
      allTaxRules = (taxRes as any).data ?? [];
    } catch (e) {
      // non-fatal
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
      await shippingApi.create({
        active: form.active,
        price_gross: priceMode === 'gross' ? Number(enteredPrice) : 0,
        price_net: priceMode === 'net' ? Number(enteredPrice) : 0,
        tax_rule_id: selectedTaxRuleId || null,
        translations: translationsToArray(translations),
      } as any);
      notifications.success($t('shipping.created'));
      goto(`${base}/shipping`);
    } catch (e) {
      notifications.error($t('common.createFailed'));
    } finally {
      submitting = false;
    }
  }
</script>

<div class="mb-6">
  <a href="{base}/shipping" class="text-sm text-primary-600 hover:underline">&larr; {$t('common.back')}</a>
</div>

<div class="card p-6 max-w-2xl">
  <h1 class="text-xl font-bold text-gray-900 mb-6">{$t('shipping.newShipping')}</h1>

  <form onsubmit={handleSubmit} class="space-y-4">
    <div class="border border-gray-200 rounded-lg p-4">
      <h3 class="text-sm font-semibold text-gray-700 mb-3">{$t('common.translations')}</h3>
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

    <!-- Steuerregel -->
    <div>
      <label class="label" for="tax_rule">{$t('products.taxRule')}</label>
      <select id="tax_rule" class="input" bind:value={selectedTaxRuleId}>
        <option value="">{$t('products.noTaxRule')}</option>
        {#each allTaxRules as t}
          <option value={t.id}>{t.name} ({t.rate / 100}%)</option>
        {/each}
      </select>
    </div>

    <!-- Preiseingabe-Modus (nur wenn Steuerregel gewählt) -->
    {#if selectedTaxRule}
      <div class="flex gap-4">
        <label class="flex items-center gap-2 cursor-pointer text-sm text-gray-700">
          <input type="radio" bind:group={priceMode} value="gross" />
          {$t('products.grossInput')}
        </label>
        <label class="flex items-center gap-2 cursor-pointer text-sm text-gray-700">
          <input type="radio" bind:group={priceMode} value="net" />
          {$t('products.netInput')}
        </label>
      </div>
    {/if}

    <!-- Preisfeld -->
    <div>
      <label class="label" for="entered_price">
        {selectedTaxRule ? (priceMode === 'gross' ? $t('products.grossLabel') : $t('products.netLabel')) : $t('products.priceGrossLabel')} {$t('products.priceCents')}
      </label>
      <input id="entered_price" class="input" type="number" min="0" bind:value={enteredPrice}
        placeholder={$t('products.grossPlaceholder')} />
    </div>

    <!-- Berechneter Gegenpreis (readonly) -->
    {#if calculatedPrice !== null}
      <p class="text-sm text-gray-500">
        {priceMode === 'gross' ? $t('products.netCalculated') : $t('products.grossCalculated')}:
        {$fmt.price(calculatedPrice)}
      </p>
    {/if}

    <div class="flex items-center gap-2">
      <input id="active" type="checkbox" bind:checked={form.active} class="h-4 w-4 rounded border-gray-300 text-primary-600" />
      <label for="active" class="text-sm text-gray-700">{$t('common.active')}</label>
    </div>

    <div class="flex gap-3 pt-2">
      <button type="submit" class="btn btn-primary" disabled={submitting}>
        {submitting ? $t('common.saving') : $t('common.save')}
      </button>
      <a href="{base}/shipping" class="btn btn-secondary">{$t('common.cancel')}</a>
    </div>
  </form>
</div>
