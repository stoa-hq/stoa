<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { page } from '$app/stores';
  import { t } from 'svelte-i18n';
  import { discountsApi } from '$lib/api/discounts';
  import { notifications } from '$lib/stores/notifications';
  import ConfirmModal from '$lib/components/ConfirmModal.svelte';

  let id = $derived($page.params.id as string);
  let loading = $state(true);
  let submitting = $state(false);
  let showDeleteConfirm = $state(false);

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

  onMount(async () => {
    try {
      const res = await discountsApi.get(id);
      const discount = res.data;
      form = {
        code: discount.code ?? '',
        type: discount.type ?? 'percentage',
        value: discount.value ?? 0,
        min_order_value: discount.min_order_value ?? 0,
        max_uses: discount.max_uses ?? 0,
        valid_from: discount.valid_from ? discount.valid_from.substring(0, 10) : '',
        valid_until: discount.valid_until ? discount.valid_until.substring(0, 10) : '',
        active: discount.active ?? true,
      };
    } catch (e) {
      notifications.error($t('discounts.loadOneFailed'));
    } finally {
      loading = false;
    }
  });

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    submitting = true;
    try {
      await discountsApi.update(id, {
        ...form,
        type: form.type as 'fixed' | 'percentage',
        value: Number(form.value),
        min_order_value: form.min_order_value ? Number(form.min_order_value) : undefined,
        max_uses: form.max_uses ? Number(form.max_uses) : undefined,
        valid_from: form.valid_from ? form.valid_from + 'T00:00:00Z' : undefined,
        valid_until: form.valid_until ? form.valid_until + 'T00:00:00Z' : undefined,
      });
      notifications.success($t('discounts.saved'));
    } catch (e) {
      notifications.error($t('common.saveFailed'));
    } finally {
      submitting = false;
    }
  }

  async function handleDelete() {
    try {
      await discountsApi.delete(id);
      notifications.success($t('discounts.deleted'));
      goto(`${base}/discounts`);
    } catch (e) {
      notifications.error($t('common.deleteFailed'));
    }
  }
</script>

<div class="mb-6">
  <a href="{base}/discounts" class="text-sm text-primary-500 hover:text-primary-400 transition-colors">&larr; {$t('common.back')}</a>
</div>

{#if loading}
  <div class="flex items-center justify-center h-32">
    <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
  </div>
{:else}
  <div class="card p-6 max-w-2xl">
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-xl font-bold text-[var(--text)]">{$t('discounts.editDiscount')}</h1>
      <button class="btn btn-danger btn-sm" onclick={() => showDeleteConfirm = true}>{$t('common.delete')}</button>
    </div>

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
{/if}

<ConfirmModal
  open={showDeleteConfirm}
  title={$t('discounts.deleteTitle')}
  message={$t('discounts.deleteMessage')}
  onConfirm={handleDelete}
  onCancel={() => showDeleteConfirm = false}
/>
