<script lang="ts">
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { t } from 'svelte-i18n';
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
      notifications.success($t('discounts.created'));
      goto(`${base}/discounts`);
    } catch (e) {
      notifications.error($t('common.createFailed'));
    } finally {
      submitting = false;
    }
  }
</script>

<div class="mb-6">
  <a href="{base}/discounts" class="text-sm text-primary-500 hover:text-primary-400 transition-colors">&larr; {$t('common.back')}</a>
</div>

<div class="card p-6 max-w-2xl">
  <h1 class="text-xl font-bold text-[var(--text)] mb-6">{$t('discounts.newDiscount')}</h1>

  <form onsubmit={handleSubmit} class="space-y-4">
    <div>
      <label class="label" for="code">{$t('discounts.code')} *</label>
      <input id="code" class="input font-mono" type="text" bind:value={form.code} required />
    </div>

    <div>
      <label class="label" for="type">{$t('discounts.type')}</label>
      <select id="type" class="input" bind:value={form.type}>
        <option value="percentage">{$t('discounts.percentage')}</option>
        <option value="fixed">{$t('discounts.fixed')}</option>
      </select>
    </div>

    <div>
      <label class="label" for="value">
        {$t('discounts.value')} {form.type === 'percentage' ? $t('discounts.valuePercentageHint') : $t('discounts.valueFixedHint')}
      </label>
      <input id="value" class="input" type="number" min="1" bind:value={form.value} />
    </div>

    <div>
      <label class="label" for="min_order_value">{$t('discounts.minOrderValue')}</label>
      <input id="min_order_value" class="input" type="number" min="0" bind:value={form.min_order_value} placeholder={$t('discounts.minOrderValuePlaceholder')} />
    </div>

    <div>
      <label class="label" for="max_uses">{$t('discounts.maxUses')}</label>
      <input id="max_uses" class="input" type="number" min="0" bind:value={form.max_uses} placeholder={$t('discounts.maxUsesPlaceholder')} />
    </div>

    <div>
      <label class="label" for="valid_from">{$t('discounts.validFrom')}</label>
      <input id="valid_from" class="input" type="date" bind:value={form.valid_from} />
    </div>

    <div>
      <label class="label" for="valid_until">{$t('discounts.validUntil')}</label>
      <input id="valid_until" class="input" type="date" bind:value={form.valid_until} />
    </div>

    <div class="flex items-center gap-2">
      <input id="active" type="checkbox" bind:checked={form.active} class="h-4 w-4 rounded border-gray-300 text-primary-600" />
      <label for="active" class="text-sm text-[var(--text-muted)]">{$t('common.active')}</label>
    </div>

    <div class="flex gap-3 pt-2">
      <button type="submit" class="btn btn-primary" disabled={submitting}>
        {submitting ? $t('common.saving') : $t('common.save')}
      </button>
      <a href="{base}/discounts" class="btn btn-secondary">{$t('common.cancel')}</a>
    </div>
  </form>
</div>
