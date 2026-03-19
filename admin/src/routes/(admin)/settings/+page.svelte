<script lang="ts">
  import { onMount } from 'svelte';
  import { t } from 'svelte-i18n';
  import { settingsApi, type UpdateSettingsRequest } from '$lib/api/settings';
  import { mediaApi } from '$lib/api/media';
  import { notifications } from '$lib/stores/notifications';
  import PluginSlot from '$lib/components/PluginSlot.svelte';

  let loading = $state(true);
  let submitting = $state(false);
  let uploadingLogo = $state(false);
  let uploadingFavicon = $state(false);

  let form: UpdateSettingsRequest = $state({
    store_name: '',
    store_description: '',
    logo_url: null,
    favicon_url: null,
    contact_email: null,
    currency: 'EUR',
    country: null,
    timezone: 'UTC',
    copyright_text: '',
    maintenance_mode: false,
  });

  onMount(async () => {
    try {
      const res = await settingsApi.get();
      const s = res.data;
      form = {
        store_name: s.store_name,
        store_description: s.store_description,
        logo_url: s.logo_url,
        favicon_url: s.favicon_url,
        contact_email: s.contact_email,
        currency: s.currency,
        country: s.country,
        timezone: s.timezone,
        copyright_text: s.copyright_text,
        maintenance_mode: s.maintenance_mode,
      };
    } catch {
      notifications.error($t('settings.loadFailed'));
    } finally {
      loading = false;
    }
  });

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    submitting = true;
    try {
      await settingsApi.update(form);
      notifications.success($t('settings.saved'));
    } catch {
      notifications.error($t('common.saveFailed'));
    } finally {
      submitting = false;
    }
  }

  async function handleImageUpload(e: Event, field: 'logo_url' | 'favicon_url') {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;

    if (field === 'logo_url') uploadingLogo = true;
    else uploadingFavicon = true;

    try {
      const res = await mediaApi.upload(file);
      form[field] = res.data.url ?? res.data.storage_path;
    } catch {
      notifications.error($t('media.uploadFailed'));
    } finally {
      if (field === 'logo_url') uploadingLogo = false;
      else uploadingFavicon = false;
      input.value = '';
    }
  }
</script>

{#if loading}
  <div class="flex items-center justify-center h-32">
    <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
  </div>
{:else}
  <form onsubmit={handleSubmit} class="max-w-2xl space-y-6">
    <h1 class="text-xl font-bold text-[var(--text)]">{$t('settings.title')}</h1>

    <!-- General -->
    <div class="card p-6">
      <h2 class="text-sm font-semibold uppercase tracking-wider text-[var(--text-muted)] mb-4">{$t('settings.general')}</h2>
      <div class="space-y-4">
        <div>
          <label class="label" for="store_name">{$t('settings.storeName')} *</label>
          <input id="store_name" class="input" type="text" bind:value={form.store_name} required maxlength="255" />
        </div>
        <div>
          <label class="label" for="store_description">{$t('settings.storeDescription')}</label>
          <textarea id="store_description" class="input" rows="3" bind:value={form.store_description} maxlength="1000"></textarea>
        </div>
        <div>
          <label class="label" for="contact_email">{$t('settings.contactEmail')}</label>
          <input id="contact_email" class="input" type="email" bind:value={form.contact_email} />
        </div>
        <div>
          <label class="label" for="copyright_text">{$t('settings.copyrightText')}</label>
          <input id="copyright_text" class="input" type="text" bind:value={form.copyright_text} maxlength="500" placeholder={$t('settings.copyrightPlaceholder')} />
          <p class="text-xs text-[var(--text-muted)] mt-1">{$t('settings.copyrightHint')}</p>
        </div>
      </div>
    </div>

    <!-- Branding -->
    <div class="card p-6">
      <h2 class="text-sm font-semibold uppercase tracking-wider text-[var(--text-muted)] mb-4">{$t('settings.branding')}</h2>
      <div class="space-y-5">
        <!-- Logo -->
        <div>
          <span class="label">{$t('settings.logo')}</span>
          <div class="flex items-start gap-4">
            {#if form.logo_url}
              <div class="shrink-0 relative group">
                <img src={form.logo_url} alt="Logo" class="h-16 w-16 rounded-lg border border-[var(--card-border)] object-contain bg-white" />
                <button
                  type="button"
                  class="absolute -top-2 -right-2 h-5 w-5 rounded-full bg-red-500 text-white text-xs flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity"
                  onclick={() => form.logo_url = null}
                  aria-label={$t('products.remove')}
                >&times;</button>
              </div>
            {/if}
            <label class="btn btn-secondary cursor-pointer {uploadingLogo ? 'opacity-50 pointer-events-none' : ''}">
              {uploadingLogo ? $t('media.uploading') : $t('settings.uploadImage')}
              <input type="file" accept="image/*" class="hidden" onchange={(e) => handleImageUpload(e, 'logo_url')} />
            </label>
          </div>
        </div>
        <!-- Favicon -->
        <div>
          <span class="label">{$t('settings.favicon')}</span>
          <div class="flex items-start gap-4">
            {#if form.favicon_url}
              <div class="shrink-0 relative group">
                <img src={form.favicon_url} alt="Favicon" class="h-10 w-10 rounded-lg border border-[var(--card-border)] object-contain bg-white" />
                <button
                  type="button"
                  class="absolute -top-2 -right-2 h-5 w-5 rounded-full bg-red-500 text-white text-xs flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity"
                  onclick={() => form.favicon_url = null}
                  aria-label={$t('products.remove')}
                >&times;</button>
              </div>
            {/if}
            <label class="btn btn-secondary cursor-pointer {uploadingFavicon ? 'opacity-50 pointer-events-none' : ''}">
              {uploadingFavicon ? $t('media.uploading') : $t('settings.uploadImage')}
              <input type="file" accept="image/*" class="hidden" onchange={(e) => handleImageUpload(e, 'favicon_url')} />
            </label>
          </div>
        </div>
      </div>
    </div>

    <!-- Regional -->
    <div class="card p-6">
      <h2 class="text-sm font-semibold uppercase tracking-wider text-[var(--text-muted)] mb-4">{$t('settings.regional')}</h2>
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div>
          <label class="label" for="currency">{$t('settings.currency')}</label>
          <input id="currency" class="input" type="text" bind:value={form.currency} maxlength="3" placeholder="EUR" required />
        </div>
        <div>
          <label class="label" for="country">{$t('settings.country')}</label>
          <input id="country" class="input" type="text" bind:value={form.country} maxlength="2" placeholder="DE" />
        </div>
        <div>
          <label class="label" for="timezone">{$t('settings.timezone')}</label>
          <input id="timezone" class="input" type="text" bind:value={form.timezone} placeholder="Europe/Berlin" required />
        </div>
      </div>
    </div>

    <!-- Advanced -->
    <div class="card p-6">
      <h2 class="text-sm font-semibold uppercase tracking-wider text-[var(--text-muted)] mb-4">{$t('settings.advanced')}</h2>
      <div class="flex items-center justify-between">
        <div>
          <span class="text-sm font-medium text-[var(--text)]">{$t('settings.maintenanceMode')}</span>
          <p class="text-xs text-[var(--text-muted)] mt-0.5">{$t('settings.maintenanceModeHint')}</p>
        </div>
        <button
          type="button"
          role="switch"
          aria-checked={form.maintenance_mode}
          aria-label={$t('settings.maintenanceMode')}
          onclick={() => form.maintenance_mode = !form.maintenance_mode}
          class="relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2
            {form.maintenance_mode ? 'bg-primary-600' : 'bg-gray-300 dark:bg-gray-600'}"
        >
          <span
            class="pointer-events-none inline-block h-5 w-5 rounded-full bg-white shadow ring-0 transition-transform duration-200
              {form.maintenance_mode ? 'translate-x-5' : 'translate-x-0'}"
          ></span>
        </button>
      </div>
    </div>

    <!-- Save -->
    <div class="flex gap-3">
      <button type="submit" class="btn btn-primary" disabled={submitting}>
        {submitting ? $t('common.saving') : $t('common.save')}
      </button>
    </div>
  </form>

  <!-- Plugin extensions -->
  <div class="max-w-2xl mt-6">
    <PluginSlot slot="admin:settings:plugins" />
  </div>
{/if}
