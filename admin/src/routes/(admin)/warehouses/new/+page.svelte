<script lang="ts">
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { t } from 'svelte-i18n';
  import { warehousesApi } from '$lib/api/warehouses';
  import { notifications } from '$lib/stores/notifications';

  let submitting = $state(false);
  let form = $state({
    name: '',
    code: '',
    active: true,
    allow_negative_stock: false,
    priority: 0,
    address_line1: '',
    address_line2: '',
    city: '',
    state: '',
    postal_code: '',
    country: '',
  });

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    if (!form.name.trim() || !form.code.trim()) {
      notifications.error($t('warehouses.nameAndCodeRequired'));
      return;
    }
    submitting = true;
    try {
      await warehousesApi.create(form);
      notifications.success($t('warehouses.created'));
      goto(`${base}/warehouses`);
    } catch (e) {
      notifications.error($t('common.createFailed'));
    } finally {
      submitting = false;
    }
  }
</script>

<div class="mb-6">
  <a href="{base}/warehouses" class="text-sm text-primary-500 hover:text-primary-400 transition-colors">&larr; {$t('common.back')}</a>
</div>

<div class="card p-6 max-w-2xl">
  <h1 class="text-xl font-bold text-[var(--text)] mb-6">{$t('warehouses.newWarehouse')}</h1>

  <form onsubmit={handleSubmit} class="space-y-4">
    <div class="grid grid-cols-2 gap-4">
      <div>
        <label class="label" for="name">{$t('common.name')}</label>
        <input id="name" class="input" type="text" bind:value={form.name} required />
      </div>
      <div>
        <label class="label" for="code">{$t('warehouses.code')}</label>
        <input id="code" class="input" type="text" bind:value={form.code} required placeholder="e.g. WH-MAIN" />
      </div>
    </div>

    <div>
      <label class="label" for="priority">{$t('warehouses.priority')}</label>
      <input id="priority" class="input" type="number" min="0" bind:value={form.priority} />
      <p class="text-xs text-[var(--text-muted)] mt-1">{$t('warehouses.priorityHint')}</p>
    </div>

    <div class="border border-[var(--card-border)] rounded-lg p-4">
      <h3 class="text-sm font-semibold text-[var(--text-muted)] mb-3">{$t('warehouses.address')}</h3>
      <div class="space-y-3">
        <div>
          <label class="label" for="address_line1">{$t('warehouses.addressLine1')}</label>
          <input id="address_line1" class="input" type="text" bind:value={form.address_line1} />
        </div>
        <div>
          <label class="label" for="address_line2">{$t('warehouses.addressLine2')}</label>
          <input id="address_line2" class="input" type="text" bind:value={form.address_line2} />
        </div>
        <div class="grid grid-cols-3 gap-3">
          <div>
            <label class="label" for="postal_code">{$t('warehouses.postalCode')}</label>
            <input id="postal_code" class="input" type="text" bind:value={form.postal_code} />
          </div>
          <div>
            <label class="label" for="city">{$t('warehouses.city')}</label>
            <input id="city" class="input" type="text" bind:value={form.city} />
          </div>
          <div>
            <label class="label" for="country">{$t('warehouses.country')}</label>
            <input id="country" class="input" type="text" bind:value={form.country} maxlength="2" placeholder="DE" />
          </div>
        </div>
      </div>
    </div>

    <div class="flex flex-col gap-2">
      <div class="flex items-center gap-2">
        <input id="active" type="checkbox" bind:checked={form.active} class="h-4 w-4 rounded border-gray-300 text-primary-600" />
        <label for="active" class="text-sm text-[var(--text-muted)]">{$t('common.active')}</label>
      </div>
      <div class="flex items-center gap-2">
        <input id="allow_negative_stock" type="checkbox" bind:checked={form.allow_negative_stock} class="h-4 w-4 rounded border-gray-300 text-primary-600" />
        <div>
          <label for="allow_negative_stock" class="text-sm text-[var(--text-muted)]">{$t('warehouses.allow_negative_stock')}</label>
          <p class="text-xs text-[var(--text-muted)]">{$t('warehouses.allow_negative_stock_hint')}</p>
        </div>
      </div>
    </div>

    <div class="flex gap-3 pt-2">
      <button type="submit" class="btn btn-primary" disabled={submitting}>
        {submitting ? $t('common.creating') : $t('common.create')}
      </button>
      <a href="{base}/warehouses" class="btn btn-secondary">{$t('common.cancel')}</a>
    </div>
  </form>
</div>
