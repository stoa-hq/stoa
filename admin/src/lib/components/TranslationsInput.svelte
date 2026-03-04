<script lang="ts">
  export interface TranslationField {
    key: string;
    label: string;
    type: 'input' | 'textarea';
    required?: boolean;
    rows?: number;
    placeholder?: string;
  }

  let {
    locales,
    localeLabels,
    primaryLocale,
    fields,
    value = $bindable(),
  }: {
    locales: string[];
    localeLabels: Record<string, string>;
    primaryLocale: string;
    fields: TranslationField[];
    value: Record<string, Record<string, string>>;
  } = $props();

  // Use an index so $state doesn't capture a reactive prop reference
  let activeLocaleIndex = $state(0);
  const activeLocale = $derived(locales[activeLocaleIndex] ?? locales[0]);

  const hasSlugField = $derived(fields.some((f) => f.key === 'slug'));

  function toSlug(name: string): string {
    return name
      .toLowerCase()
      .trim()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-|-$/g, '');
  }

  function handleInput(locale: string, fieldKey: string, fieldValue: string) {
    value[locale][fieldKey] = fieldValue;
    if (fieldKey === 'name' && hasSlugField && !value[locale].slug) {
      value[locale].slug = toSlug(fieldValue);
    }
  }
</script>

<div>
  <!-- Locale Tabs -->
  <div class="flex border-b border-gray-200 mb-4">
    {#each locales as locale, i}
      {@const isActive = activeLocale === locale}
      {@const isPrimary = locale === primaryLocale}
      <button
        type="button"
        class="px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors {isActive
          ? 'border-primary-600 text-primary-600'
          : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'}"
        onclick={() => (activeLocaleIndex = i)}
      >
        {localeLabels[locale] ?? locale}
        {#if isPrimary}
          <span class="text-primary-400 ml-1 text-xs">●</span>
        {/if}
      </button>
    {/each}
  </div>

  <!-- Fields for active locale -->
  <div class="space-y-4">
    {#each fields as field}
      {@const isRequired = !!field.required && activeLocale === primaryLocale}
      <div>
        <label class="label" for="{activeLocale}-{field.key}">
          {field.label}{#if field.required}<span class="text-red-500 ml-0.5">*</span>{/if}
        </label>
        {#if field.type === 'textarea'}
          <textarea
            id="{activeLocale}-{field.key}"
            class="input"
            rows={field.rows ?? 4}
            value={value[activeLocale]?.[field.key] ?? ''}
            oninput={(e) => handleInput(activeLocale, field.key, e.currentTarget.value)}
            required={isRequired}
            placeholder={field.placeholder}
          ></textarea>
        {:else}
          <input
            id="{activeLocale}-{field.key}"
            class="input"
            type="text"
            value={value[activeLocale]?.[field.key] ?? ''}
            oninput={(e) => handleInput(activeLocale, field.key, e.currentTarget.value)}
            required={isRequired}
            placeholder={field.placeholder}
          />
        {/if}
      </div>
    {/each}
  </div>
</div>
