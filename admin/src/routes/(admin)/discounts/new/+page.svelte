<script lang="ts">
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { discountsApi } from '$lib/api/discounts';
  import { notifications } from '$lib/stores/notifications';

  let submitting = $state(false);
  let form = $state({
    code: '',
    type: 'percentage',
    value: 0,
    min_order_value: 0,
    max_uses: 0,
    valid_from: '',
    valid_until: '',
    active: true,
  });

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    submitting = true;
    try {
      await discountsApi.create({
        ...form,
        type: form.type as 'fixed' | 'percentage',
        value: Number(form.value),
        min_order_value: form.min_order_value ? Number(form.min_order_value) : undefined,
        max_uses: form.max_uses ? Number(form.max_uses) : undefined,
        valid_from: form.valid_from ? form.valid_from + 'T00:00:00Z' : undefined,
        valid_until: form.valid_until ? form.valid_until + 'T00:00:00Z' : undefined,
      });
      notifications.success('Rabatt erstellt.');
      goto(`${base}/discounts`);
    } catch (e) {
      notifications.error('Erstellen fehlgeschlagen.');
    } finally {
      submitting = false;
    }
  }
</script>

<div class="mb-6">
  <a href="{base}/discounts" class="text-sm text-primary-600 hover:underline">← Zurück</a>
</div>

<div class="card p-6 max-w-2xl">
  <h1 class="text-xl font-bold text-gray-900 mb-6">Neuer Rabatt</h1>

  <form onsubmit={handleSubmit} class="space-y-4">
    <div>
      <label class="label" for="code">Code *</label>
      <input id="code" class="input font-mono" type="text" bind:value={form.code} required />
    </div>

    <div>
      <label class="label" for="type">Typ</label>
      <select id="type" class="input" bind:value={form.type}>
        <option value="percentage">Prozent</option>
        <option value="fixed">Fixbetrag</option>
      </select>
    </div>

    <div>
      <label class="label" for="value">
        Wert {form.type === 'percentage' ? '(Basispunkte, z.B. 1000 = 10%)' : '(Cent, z.B. 500 = 5,00 €)'}
      </label>
      <input id="value" class="input" type="number" min="1" bind:value={form.value} />
    </div>

    <div>
      <label class="label" for="min_order_value">Mindestbestellwert (Cent)</label>
      <input id="min_order_value" class="input" type="number" min="0" bind:value={form.min_order_value} placeholder="0 = kein Minimum" />
    </div>

    <div>
      <label class="label" for="max_uses">Max. Verwendungen</label>
      <input id="max_uses" class="input" type="number" min="0" bind:value={form.max_uses} placeholder="0 = unbegrenzt" />
    </div>

    <div>
      <label class="label" for="valid_from">Gültig ab</label>
      <input id="valid_from" class="input" type="date" bind:value={form.valid_from} />
    </div>

    <div>
      <label class="label" for="valid_until">Gültig bis</label>
      <input id="valid_until" class="input" type="date" bind:value={form.valid_until} />
    </div>

    <div class="flex items-center gap-2">
      <input id="active" type="checkbox" bind:checked={form.active} class="h-4 w-4 rounded border-gray-300 text-primary-600" />
      <label for="active" class="text-sm text-gray-700">Aktiv</label>
    </div>

    <div class="flex gap-3 pt-2">
      <button type="submit" class="btn btn-primary" disabled={submitting}>
        {submitting ? 'Speichern...' : 'Speichern'}
      </button>
      <a href="{base}/discounts" class="btn btn-secondary">Abbrechen</a>
    </div>
  </form>
</div>
