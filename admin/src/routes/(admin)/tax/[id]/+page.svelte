<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { page } from '$app/stores';
  import { taxApi } from '$lib/api/tax';
  import { notifications } from '$lib/stores/notifications';
  import ConfirmModal from '$lib/components/ConfirmModal.svelte';

  let id = $derived($page.params.id as string);
  let loading = $state(true);
  let submitting = $state(false);
  let showDeleteConfirm = $state(false);

  let form = $state({
    name: '',
    rate: 0,
    country_code: '',
    type: 'standard',
  });

  onMount(async () => {
    try {
      const res = await taxApi.get(id);
      const rule = res.data;
      form = {
        name: rule.name ?? '',
        rate: rule.rate ?? 0,
        country_code: rule.country_code ?? '',
        type: rule.type ?? 'standard',
      };
    } catch (e) {
      notifications.error('Steuerregel konnte nicht geladen werden.');
    } finally {
      loading = false;
    }
  });

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    submitting = true;
    try {
      await taxApi.update(id, {
        ...form,
        rate: Number(form.rate),
        country_code: form.country_code || undefined,
      });
      notifications.success('Steuerregel gespeichert.');
    } catch (e) {
      notifications.error('Speichern fehlgeschlagen.');
    } finally {
      submitting = false;
    }
  }

  async function handleDelete() {
    try {
      await taxApi.delete(id);
      notifications.success('Steuerregel gelöscht.');
      goto(`${base}/tax`);
    } catch (e) {
      notifications.error('Löschen fehlgeschlagen.');
    }
  }
</script>

<div class="mb-6">
  <a href="{base}/tax" class="text-sm text-primary-600 hover:underline">← Zurück</a>
</div>

{#if loading}
  <div class="flex items-center justify-center h-32">
    <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
  </div>
{:else}
  <div class="card p-6 max-w-2xl">
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-xl font-bold text-gray-900">Steuerregel bearbeiten</h1>
      <button class="btn btn-danger btn-sm" onclick={() => showDeleteConfirm = true}>Löschen</button>
    </div>

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
{/if}

<ConfirmModal
  open={showDeleteConfirm}
  title="Steuerregel löschen"
  message="Soll diese Steuerregel wirklich gelöscht werden?"
  onConfirm={handleDelete}
  onCancel={() => showDeleteConfirm = false}
/>
