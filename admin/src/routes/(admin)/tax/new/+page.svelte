<script lang="ts">
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { taxApi } from '$lib/api/tax';
  import { notifications } from '$lib/stores/notifications';

  let submitting = $state(false);
  let form = $state({
    name: '',
    rate: 0,
    country_code: '',
    type: 'standard',
  });

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    submitting = true;
    try {
      await taxApi.create({
        ...form,
        rate: Number(form.rate),
        country_code: form.country_code || undefined,
      });
      notifications.success('Steuerregel erstellt.');
      goto(`${base}/tax`);
    } catch (e) {
      notifications.error('Erstellen fehlgeschlagen.');
    } finally {
      submitting = false;
    }
  }
</script>

<div class="mb-6">
  <a href="{base}/tax" class="text-sm text-primary-600 hover:underline">← Zurück</a>
</div>

<div class="card p-6 max-w-2xl">
  <h1 class="text-xl font-bold text-gray-900 mb-6">Neue Steuerregel</h1>

  <form onsubmit={handleSubmit} class="space-y-4">
    <div>
      <label class="label" for="name">Name *</label>
      <input id="name" class="input" type="text" bind:value={form.name} required />
    </div>

    <div>
      <label class="label" for="rate">Rate in Basispunkten</label>
      <input id="rate" class="input" type="number" min="0" bind:value={form.rate} placeholder="1900 = 19%" />
    </div>

    <div>
      <label class="label" for="country_code">Ländercode</label>
      <input id="country_code" class="input" type="text" bind:value={form.country_code} placeholder="z.B. DE" maxlength="2" />
    </div>

    <div>
      <label class="label" for="type">Typ</label>
      <select id="type" class="input" bind:value={form.type}>
        <option value="standard">Standard</option>
        <option value="reduced">Ermäßigt</option>
        <option value="zero">Nullsatz</option>
      </select>
    </div>

    <div class="flex gap-3 pt-2">
      <button type="submit" class="btn btn-primary" disabled={submitting}>
        {submitting ? 'Speichern...' : 'Speichern'}
      </button>
      <a href="{base}/tax" class="btn btn-secondary">Abbrechen</a>
    </div>
  </form>
</div>
