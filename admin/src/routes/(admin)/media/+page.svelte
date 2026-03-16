<script lang="ts">
  import { onMount } from 'svelte';
  import { t } from 'svelte-i18n';
  import { mediaApi } from '$lib/api/media';
  import { notifications } from '$lib/stores/notifications';
  import { formatBytes } from '$lib/utils';
  import ConfirmModal from '$lib/components/ConfirmModal.svelte';
  import Skeleton from '$lib/components/Skeleton.svelte';
  import { Upload, FileText, X } from 'lucide-svelte';

  let items = $state<any[]>([]);
  let loading = $state(true);
  let uploading = $state(false);
  let deleteId = $state<string | null>(null);
  let showConfirm = $state(false);
  let dragOver = $state(false);
  let fileInput: HTMLInputElement;

  async function load() {
    loading = true;
    try {
      const res = await mediaApi.list({ limit: 100 });
      items = res.data ?? [];
    } catch (e) {
      notifications.error($t('media.loadFailed'));
    } finally {
      loading = false;
    }
  }

  onMount(load);

  async function uploadFiles(files: FileList | File[]) {
    if (!files || files.length === 0) return;
    uploading = true;
    try {
      for (const file of Array.from(files)) {
        await mediaApi.upload(file);
      }
      notifications.success($t('media.uploadSuccess'));
      load();
    } catch (e) {
      notifications.error($t('media.uploadFailed'));
    } finally {
      uploading = false;
    }
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    dragOver = false;
    if (e.dataTransfer?.files) {
      uploadFiles(e.dataTransfer.files);
    }
  }

  function handleFileInput(e: Event) {
    const input = e.target as HTMLInputElement;
    if (input.files) {
      uploadFiles(input.files);
    }
  }

  function confirmDelete(id: string) {
    deleteId = id;
    showConfirm = true;
  }

  async function doDelete() {
    if (!deleteId) return;
    try {
      await mediaApi.delete(deleteId);
      notifications.success($t('media.fileDeleted'));
      showConfirm = false;
      deleteId = null;
      load();
    } catch (e) {
      notifications.error($t('common.deleteFailed'));
    }
  }

  function isImage(item: any) {
    return item.mime_type?.startsWith('image/') ?? false;
  }
</script>

<div class="flex items-center justify-between mb-6">
  <h1 class="text-2xl font-bold text-[var(--text)]">{$t('media.title')}</h1>
</div>

<!-- Upload Area -->
<div
  class="card p-6 mb-6 border-2 border-dashed transition-colors {dragOver ? 'border-primary-400 bg-primary-50 dark:bg-primary-900/10' : 'border-[var(--card-border)]'}"
  ondragover={(e) => { e.preventDefault(); dragOver = true; }}
  ondragleave={() => dragOver = false}
  ondrop={handleDrop}
  role="button"
  tabindex="0"
  onclick={() => fileInput.click()}
  onkeydown={(e) => e.key === 'Enter' && fileInput.click()}
>
  <div class="text-center">
    <Upload class="mx-auto h-10 w-10 text-[var(--text-muted)] mb-2" />
    <p class="text-sm text-[var(--text-muted)]">
      {#if uploading}
        {$t('media.uploading')}
      {:else}
        {$t('media.dropOrClick', { values: { click: '' } }).replace('{click}', '')}<span class="text-primary-500 font-medium">{$t('media.clickToSelect')}</span>
      {/if}
    </p>
  </div>
  <input
    bind:this={fileInput}
    type="file"
    multiple
    class="hidden"
    onchange={handleFileInput}
  />
</div>

<!-- Media Grid -->
<div class="card p-6">
  {#if loading}
    <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-4">
      {#each Array(6) as _}
        <div class="skeleton aspect-square rounded-lg"></div>
      {/each}
    </div>
  {:else if items.length === 0}
    <p class="text-center text-[var(--text-muted)] text-sm py-8">{$t('media.noMedia')}</p>
  {:else}
    <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-4">
      {#each items as item}
        <div class="group relative">
          <div class="aspect-square rounded-lg overflow-hidden bg-gray-100 dark:bg-gray-800 flex items-center justify-center">
            {#if isImage(item)}
              <img src={item.url} alt={item.filename} class="w-full h-full object-cover" />
            {:else}
              <FileText class="h-10 w-10 text-[var(--text-muted)]" />
            {/if}
          </div>
          <div class="mt-1">
            <p class="text-xs text-[var(--text)] truncate" title={item.filename}>{item.filename}</p>
            <p class="text-xs text-[var(--text-muted)]">{formatBytes(item.size)}</p>
          </div>
          <button
            class="absolute top-1 right-1 opacity-0 group-hover:opacity-100 transition-opacity p-1 rounded-md bg-red-600 text-white hover:bg-red-700"
            onclick={() => confirmDelete(item.id)}
            title={$t('common.delete')}
          >
            <X class="h-3 w-3" />
          </button>
        </div>
      {/each}
    </div>
  {/if}
</div>

<ConfirmModal
  open={showConfirm}
  title={$t('media.deleteTitle')}
  message={$t('media.deleteMessage')}
  onConfirm={doDelete}
  onCancel={() => { showConfirm = false; deleteId = null; }}
/>
