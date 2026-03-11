<script lang="ts">
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { t } from 'svelte-i18n';
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
      notifications.success($t('tax.created'));
      goto(`${base}/tax`);
    } catch (e) {
      notifications.error($t('common.createFailed'));
    } finally {
      submitting = false;
    }
  }
</script>

<div class="mb-6">
  <a href="{base}/tax" class="text-sm text-primary-600 hover:underline">&larr; {$t('common.back')}</a>
</div>

<div class="card p-6 max-w-2xl">
  <h1 class="text-xl font-bold text-gray-900 mb-6">{$t('tax.newTaxRule')}</h1>

  <form onsubmit={handleSubmit} class="space-y-4">
    <div>
      <label class="label" for="name">{$t('common.name')} *</label>
      <input id="name" class="input" type="text" bind:value={form.name} required />
    </div>

    <div>
      <label class="label" for="rate">{$t('tax.rateInBasisPoints')}</label>
      <input id="rate" class="input" type="number" min="0" bind:value={form.rate} placeholder={$t('tax.ratePlaceholder')} />
    </div>

    <div>
      <label class="label" for="country_code">{$t('tax.countryCode')}</label>
      <input id="country_code" class="input" type="text" bind:value={form.country_code} placeholder={$t('tax.countryCodePlaceholder')} maxlength="2" />
    </div>

    <div>
      <label class="label" for="type">{$t('common.type')}</label>
      <select id="type" class="input" bind:value={form.type}>
        <option value="standard">{$t('tax.standard')}</option>
        <option value="reduced">{$t('tax.reduced')}</option>
        <option value="zero">{$t('tax.zero')}</option>
      </select>
    </div>

    <div class="flex gap-3 pt-2">
      <button type="submit" class="btn btn-primary" disabled={submitting}>
        {submitting ? $t('common.saving') : $t('common.save')}
      </button>
      <a href="{base}/tax" class="btn btn-secondary">{$t('common.cancel')}</a>
    </div>
  </form>
</div>
