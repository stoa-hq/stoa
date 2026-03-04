<script lang="ts">
  import { onMount } from 'svelte';
  import { tagsApi } from '$lib/api/tags';
  import { notifications } from '$lib/stores/notifications';
  import Modal from '$lib/components/Modal.svelte';
  import ConfirmModal from '$lib/components/ConfirmModal.svelte';

  let items = $state<any[]>([]);
  let loading = $state(true);
  let deleteId = $state<string | null>(null);
  let showConfirm = $state(false);
  let showModal = $state(false);
  let editItem = $state<any | null>(null);
  let modalSubmitting = $state(false);

  let newTagForm = $state({ name: '', slug: '' });
  let modalForm = $state({ name: '', slug: '' });

  async function load() {
    loading = true;
    try {
      const res = await tagsApi.list({ limit: 200 });
      items = res.data ?? [];
    } catch (e) {
      notifications.error('Tags konnten nicht geladen werden.');
    } finally {
      loading = false;
    }
  }

  onMount(load);

  async function handleCreate(e: SubmitEvent) {
    e.preventDefault();
    try {
      await tagsApi.create({ ...newTagForm });
      notifications.success('Tag erstellt.');
      newTagForm = { name: '', slug: '' };
      load();
    } catch (e) {
      notifications.error('Erstellen fehlgeschlagen.');
    }
  }

  function openEdit(item: any) {
    editItem = item;
    modalForm = { name: item.name ?? '', slug: item.slug ?? '' };
    showModal = true;
  }

  async function handleModalSubmit(e: SubmitEvent) {
    e.preventDefault();
    if (!editItem) return;
    modalSubmitting = true;
    try {
      await tagsApi.update(editItem.id, { ...modalForm });
      notifications.success('Tag gespeichert.');
      showModal = false;
      editItem = null;
      load();
    } catch (e) {
      notifications.error('Speichern fehlgeschlagen.');
    } finally {
      modalSubmitting = false;
    }
  }

  function confirmDelete(id: string, e: MouseEvent) {
    e.stopPropagation();
    deleteId = id;
    showConfirm = true;
  }

  async function doDelete() {
    if (!deleteId) return;
    try {
      await tagsApi.delete(deleteId);
      notifications.success('Tag gelöscht.');
      showConfirm = false;
      deleteId = null;
      load();
    } catch (e) {
      notifications.error('Löschen fehlgeschlagen.');
    }
  }
</script>

<div class="flex items-center justify-between mb-6">
  <h1 class="text-2xl font-bold text-gray-900">Tags</h1>
</div>

<!-- Inline Create Form -->
<div class="card p-6 mb-6">
  <h2 class="text-base font-semibold text-gray-900 mb-3">Neuer Tag</h2>
  <form onsubmit={handleCreate} class="flex gap-3 items-end">
    <div class="flex-1">
      <label class="label" for="new-name">Name *</label>
      <input id="new-name" class="input" type="text" bind:value={newTagForm.name} required placeholder="Tag-Name" />
    </div>
    <div class="flex-1">
      <label class="label" for="new-slug">Slug</label>
      <input id="new-slug" class="input" type="text" bind:value={newTagForm.slug} placeholder="tag-slug" />
    </div>
    <button type="submit" class="btn btn-primary">Hinzufügen</button>
  </form>
</div>

<!-- Tags Table -->
<div class="card p-6">
  {#if loading}
    <div class="flex items-center justify-center h-32">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
    </div>
  {:else}
    <div class="overflow-x-auto">
      <table class="min-w-full divide-y divide-gray-200">
        <thead>
          <tr>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Name</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Slug</th>
            <th class="px-4 py-3"></th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          {#each items as item}
            <tr class="hover:bg-gray-50 cursor-pointer" onclick={() => openEdit(item)}>
              <td class="px-4 py-3 text-sm font-medium text-gray-900">{item.name}</td>
              <td class="px-4 py-3 text-sm text-gray-600">{item.slug}</td>
              <td class="px-4 py-3 text-right">
                <button class="btn btn-danger btn-sm" onclick={(e) => confirmDelete(item.id, e)}>Löschen</button>
              </td>
            </tr>
          {/each}
          {#if items.length === 0}
            <tr>
              <td colspan="3" class="px-4 py-6 text-center text-gray-400 text-sm">Keine Tags gefunden.</td>
            </tr>
          {/if}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<Modal open={showModal} title="Tag bearbeiten" onClose={() => { showModal = false; editItem = null; }}>
  <form onsubmit={handleModalSubmit} class="space-y-4">
    <div>
      <label class="label" for="edit-name">Name *</label>
      <input id="edit-name" class="input" type="text" bind:value={modalForm.name} required />
    </div>
    <div>
      <label class="label" for="edit-slug">Slug</label>
      <input id="edit-slug" class="input" type="text" bind:value={modalForm.slug} />
    </div>
    <div class="flex gap-3 pt-2">
      <button type="submit" class="btn btn-primary" disabled={modalSubmitting}>
        {modalSubmitting ? 'Speichern...' : 'Speichern'}
      </button>
      <button type="button" class="btn btn-secondary" onclick={() => { showModal = false; editItem = null; }}>Abbrechen</button>
    </div>
  </form>
</Modal>

<ConfirmModal
  open={showConfirm}
  title="Tag löschen"
  message="Soll dieser Tag wirklich gelöscht werden?"
  onConfirm={doDelete}
  onCancel={() => { showConfirm = false; deleteId = null; }}
/>
