<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { shippingApi } from '$lib/api/shipping';
  import { taxApi } from '$lib/api/tax';
  import { notifications } from '$lib/stores/notifications';
  import { formatPrice } from '$lib/utils';
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
      notifications.error(`Bitte Name auf ${LOCALE_LABELS[DEFAULT_LOCALE]} ausfüllen.`);
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
      notifications.success('Versandmethode erstellt.');
      goto(`${base}/shipping`);
    } catch (e) {
      notifications.error('Erstellen fehlgeschlagen.');
    } finally {
      submitting = false;
    }
  }
</script>

<div class="mb-6">
  <a href="{base}/shipping" class="text-sm text-primary-600 hover:underline">← Zurück</a>
</div>

<div class="card p-6 max-w-2xl">
  <h1 class="text-xl font-bold text-gray-900 mb-6">Neue Versandmethode</h1>

  <form onsubmit={handleSubmit} class="space-y-4">
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

    <!-- Steuerregel -->
    <div>
      <label class="label" for="tax_rule">Steuerregel</label>
      <select id="tax_rule" class="input" bind:value={selectedTaxRuleId}>
        <option value="">Keine Steuerregel</option>
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
          Brutto eingeben
        </label>
        <label class="flex items-center gap-2 cursor-pointer text-sm text-gray-700">
          <input type="radio" bind:group={priceMode} value="net" />
          Netto eingeben
        </label>
      </div>
    {/if}

    <!-- Preisfeld -->
    <div>
      <label class="label" for="entered_price">
        {selectedTaxRule ? (priceMode === 'gross' ? 'Brutto' : 'Netto') : 'Preis brutto'} (Cent)
      </label>
      <input id="entered_price" class="input" type="number" min="0" bind:value={enteredPrice}
        placeholder="{selectedTaxRule ? (priceMode === 'gross' ? 'Brutto' : 'Netto') : 'Brutto'} in Cent (499 = 4,99 €)" />
    </div>

    <!-- Berechneter Gegenpreis (readonly) -->
    {#if calculatedPrice !== null}
      <p class="text-sm text-gray-500">
        {priceMode === 'gross' ? 'Netto (berechnet)' : 'Brutto (berechnet)'}:
        {formatPrice(calculatedPrice)}
      </p>
    {/if}

    <div class="flex items-center gap-2">
      <input id="active" type="checkbox" bind:checked={form.active} class="h-4 w-4 rounded border-gray-300 text-primary-600" />
      <label for="active" class="text-sm text-gray-700">Aktiv</label>
    </div>

    <div class="flex gap-3 pt-2">
      <button type="submit" class="btn btn-primary" disabled={submitting}>
        {submitting ? 'Speichern...' : 'Speichern'}
      </button>
      <a href="{base}/shipping" class="btn btn-secondary">Abbrechen</a>
    </div>
  </form>
</div>
