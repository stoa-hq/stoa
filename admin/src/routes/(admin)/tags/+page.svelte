<script lang="ts">
  import { onMount } from 'svelte';
  import { t } from 'svelte-i18n';
  import { tagsApi } from '$lib/api/tags';
  import { notifications } from '$lib/stores/notifications';
  import Modal from '$lib/components/Modal.svelte';
  import ConfirmModal from '$lib/components/ConfirmModal.svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';

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
      notifications.error($t('tags.loadFailed'));
    } finally {
      loading = false;
    }
  }

  onMount(load);

  async function handleCreate(e: SubmitEvent) {
    e.preventDefault();
    try {
      await tagsApi.create({ ...newTagForm });
      notifications.success($t('tags.created'));
      newTagForm = { name: '', slug: '' };
      load();
    } catch (e) {
      notifications.error($t('common.createFailed'));
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
      notifications.success($t('tags.saved'));
      showModal = false;
      editItem = null;
      load();
    } catch (e) {
      notifications.error($t('common.saveFailed'));
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
      notifications.success($t('tags.deleted'));
      showConfirm = false;
      deleteId = null;
      load();
    } catch (e) {
      notifications.error($t('common.deleteFailed'));
    }
  }
</script>

<div class="flex items-center justify-between mb-6">
  <h1 class="text-2xl font-bold text-[var(--text)]">{$t('tags.title')}</h1>
</div>

<!-- Inline Create Form -->
<div class="card p-6 mb-6">
  <h2 class="text-base font-semibold text-[var(--text)] mb-3">{$t('tags.newTag')}</h2>
  <form onsubmit={handleCreate} class="flex flex-col sm:flex-row gap-3 sm:items-end">
    <div class="flex-1">
      <label class="label" for="new-name">{$t('common.name')} *</label>
      <input id="new-name" class="input" type="text" bind:value={newTagForm.name} required placeholder={$t('tags.tagName')} />
    </div>
    <div class="flex-1">
      <label class="label" for="new-slug">{$t('common.slug')}</label>
      <input id="new-slug" class="input" type="text" bind:value={newTagForm.slug} placeholder={$t('tags.tagSlug')} />
    </div>
    <button type="submit" class="btn btn-primary">{$t('common.add')}</button>
  </form>
</div>

<!-- Tags Table -->
<div class="card p-6">
  {#if loading}
    <div class="space-y-3">
      {#each Array(5) as _}
        <Skeleton height="h-10" />
      {/each}
    </div>
  {:else}
    <div class="overflow-x-auto">
      <table class="min-w-full divide-y divide-[var(--card-border)]">
        <thead>
          <tr>
            <th class="table-header">{$t('common.name')}</th>
            <th class="table-header">{$t('common.slug')}</th>
            <th class="table-header"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-[var(--card-border)]">
          {#each items as item}
            <tr class="table-row cursor-pointer" onclick={() => openEdit(item)}>
              <td class="table-cell font-medium text-[var(--text)]">{item.name}</td>
              <td class="table-cell text-[var(--text-muted)]">{item.slug}</td>
              <td class="table-cell text-right">
                <button class="btn btn-danger btn-sm" onclick={(e) => confirmDelete(item.id, e)}>{$t('common.delete')}</button>
              </td>
            </tr>
          {/each}
          {#if items.length === 0}
            <tr>
              <td colspan="3" class="table-cell text-center text-[var(--text-muted)] py-6">{$t('tags.noTags')}</td>
            </tr>
          {/if}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<Modal open={showModal} title={$t('tags.editTag')} onClose={() => { showModal = false; editItem = null; }}>
  <form onsubmit={handleModalSubmit} class="space-y-4">
    <div>
      <label class="label" for="edit-name">{$t('common.name')} *</label>
      <input id="edit-name" class="input" type="text" bind:value={modalForm.name} required />
    </div>
    <div>
      <label class="label" for="edit-slug">{$t('common.slug')}</label>
      <input id="edit-slug" class="input" type="text" bind:value={modalForm.slug} />
    </div>
    <div class="flex gap-3 pt-2">
      <button type="submit" class="btn btn-primary" disabled={modalSubmitting}>
        {modalSubmitting ? $t('common.saving') : $t('common.save')}
      </button>
      <button type="button" class="btn btn-secondary" onclick={() => { showModal = false; editItem = null; }}>{$t('common.cancel')}</button>
    </div>
  </form>
</Modal>

<ConfirmModal
  open={showConfirm}
  title={$t('tags.deleteTitle')}
  message={$t('tags.deleteMessage')}
  onConfirm={doDelete}
  onCancel={() => { showConfirm = false; deleteId = null; }}
/>
