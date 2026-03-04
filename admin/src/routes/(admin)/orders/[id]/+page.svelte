<script lang="ts">
  import { onMount } from 'svelte';
  import { base } from '$app/paths';
  import { page } from '$app/stores';
  import { ordersApi } from '$lib/api/orders';
  import { notifications } from '$lib/stores/notifications';
  import { formatPrice, formatDateTime, orderStatusBadge } from '$lib/utils';
  import Modal from '$lib/components/Modal.svelte';

  let id = $derived($page.params.id as string);
  let loading = $state(true);
  let order = $state<any>(null);
  let showStatusModal = $state(false);
  let statusSubmitting = $state(false);

  let statusForm = $state({
    status: '',
    comment: '',
  });

  const allStatuses: Record<string, string> = {
    pending: 'Ausstehend',
    confirmed: 'Bestätigt',
    processing: 'In Bearbeitung',
    shipped: 'Versendet',
    delivered: 'Geliefert',
    cancelled: 'Storniert',
    refunded: 'Erstattet',
  };

  const validTransitions: Record<string, string[]> = {
    pending: ['confirmed', 'cancelled'],
    confirmed: ['processing', 'cancelled'],
    processing: ['shipped', 'cancelled'],
    shipped: ['delivered'],
    delivered: ['refunded'],
    cancelled: [],
    refunded: [],
  };

  const statusOptions = $derived(
    (validTransitions[order?.status ?? ''] ?? []).map((v) => ({ value: v, label: allStatuses[v] }))
  );

  onMount(async () => {
    try {
      order = (await ordersApi.get(id)).data;
      statusForm.status = order.status ?? '';
    } catch (e) {
      notifications.error('Bestellung konnte nicht geladen werden.');
    } finally {
      loading = false;
    }
  });

  async function handleStatusSubmit(e: SubmitEvent) {
    e.preventDefault();
    statusSubmitting = true;
    try {
      await ordersApi.updateStatus(id, statusForm.status, statusForm.comment || undefined);
      notifications.success('Status aktualisiert.');
      order = (await ordersApi.get(id)).data;
      showStatusModal = false;
    } catch (e) {
      notifications.error('Status-Änderung fehlgeschlagen.');
    } finally {
      statusSubmitting = false;
    }
  }
</script>

<div class="mb-6">
  <a href="{base}/orders" class="text-sm text-primary-600 hover:underline">← Zurück</a>
</div>

{#if loading}
  <div class="flex items-center justify-center h-32">
    <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
  </div>
{:else if order}
  <!-- Order Info Card -->
  <div class="card p-6 mb-6">
    <div class="flex items-center justify-between mb-4">
      <h1 class="text-xl font-bold text-gray-900">Bestellung #{order.order_number ?? order.id}</h1>
      {#if statusOptions.length > 0}
        <button class="btn btn-secondary btn-sm" onclick={() => { statusForm.status = statusOptions[0].value; showStatusModal = true; }}>
          Status ändern
        </button>
      {/if}
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
      <div>
        <p class="text-xs text-gray-500 uppercase font-medium">Status</p>
        <span class="badge {orderStatusBadge(order.status)} mt-1">{order.status}</span>
      </div>
      <div>
        <p class="text-xs text-gray-500 uppercase font-medium">Erstellt</p>
        <p class="text-sm text-gray-900 mt-1">{formatDateTime(order.created_at)}</p>
      </div>
      <div>
        <p class="text-xs text-gray-500 uppercase font-medium">Aktualisiert</p>
        <p class="text-sm text-gray-900 mt-1">{formatDateTime(order.updated_at)}</p>
      </div>
      <div>
        <p class="text-xs text-gray-500 uppercase font-medium">Zwischensumme</p>
        <p class="text-sm text-gray-900 mt-1">{formatPrice(order.subtotal_gross)}</p>
      </div>
      <div>
        <p class="text-xs text-gray-500 uppercase font-medium">Versand</p>
        <p class="text-sm text-gray-900 mt-1">{formatPrice(order.shipping_cost)}</p>
      </div>
      <div>
        <p class="text-xs text-gray-500 uppercase font-medium">Gesamt</p>
        <p class="text-sm font-bold text-gray-900 mt-1">{formatPrice(order.total)}</p>
      </div>
    </div>
  </div>

  <!-- Order Items -->
  <div class="card p-6">
    <h2 class="text-lg font-semibold text-gray-900 mb-4">Positionen</h2>
    <div class="overflow-x-auto">
      <table class="min-w-full divide-y divide-gray-200">
        <thead>
          <tr>
            <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Produkt</th>
            <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">SKU</th>
            <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Menge</th>
            <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Einzelpreis</th>
            <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Gesamt</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-200">
          {#each order.items ?? [] as item}
            <tr>
              <td class="px-4 py-2 text-sm text-gray-900">{item.name ?? '—'}</td>
              <td class="px-4 py-2 text-sm text-gray-600">{item.sku ?? '—'}</td>
              <td class="px-4 py-2 text-sm text-gray-700">{item.quantity}</td>
              <td class="px-4 py-2 text-sm text-gray-700">{formatPrice(item.unit_price_gross)}</td>
              <td class="px-4 py-2 text-sm text-gray-700">{formatPrice(item.total_gross)}</td>
            </tr>
          {/each}
          {#if (order.items ?? []).length === 0}
            <tr>
              <td colspan="5" class="px-4 py-4 text-center text-gray-400 text-sm">Keine Positionen.</td>
            </tr>
          {/if}
        </tbody>
      </table>
    </div>
  </div>
{/if}

<Modal open={showStatusModal} title="Status ändern" onClose={() => showStatusModal = false}>
  <form onsubmit={handleStatusSubmit} class="space-y-4">
    <div>
      <label class="label" for="status">Status</label>
      <select id="status" class="input" bind:value={statusForm.status}>
        {#each statusOptions as opt}
          <option value={opt.value}>{opt.label}</option>
        {/each}
      </select>
    </div>
    <div>
      <label class="label" for="comment">Kommentar (optional)</label>
      <textarea id="comment" class="input" rows="3" bind:value={statusForm.comment}></textarea>
    </div>
    <div class="flex gap-3 pt-2">
      <button type="submit" class="btn btn-primary" disabled={statusSubmitting}>
        {statusSubmitting ? 'Speichern...' : 'Speichern'}
      </button>
      <button type="button" class="btn btn-secondary" onclick={() => showStatusModal = false}>Abbrechen</button>
    </div>
  </form>
</Modal>
