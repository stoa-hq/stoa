<script lang="ts">
	import { onMount } from 'svelte';
	import { locale } from 'svelte-i18n';
	import type { UISchema } from '$lib/stores/plugins';
	import { createPluginClient } from '$lib/api/plugin-client';

	interface Props {
		schema: UISchema;
		context?: Record<string, unknown>;
		onEvent?: (e: CustomEvent) => void;
	}

	let { schema, context = {}, onEvent }: Props = $props();

	let values = $state<Record<string, unknown>>({});
	let loading = $state(false);
	let submitting = $state(false);
	let error = $state('');

	const client = createPluginClient();
	const loc = $derived($locale ?? 'en');

	function label(i18n: Record<string, string> | undefined): string {
		if (!i18n) return '';
		return i18n[loc] ?? i18n['en'] ?? Object.values(i18n)[0] ?? '';
	}

	onMount(async () => {
		if (schema.load_url) {
			loading = true;
			try {
				const data = await client.get<Record<string, unknown>>(schema.load_url);
				values = data;
			} catch {
				// Ignore load errors — fields will be empty
			} finally {
				loading = false;
			}
		}
	});

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		if (!schema.submit_url) return;
		submitting = true;
		error = '';
		try {
			await client.post(schema.submit_url, { ...values, ...context });
			onEvent?.(new CustomEvent('plugin:submit', { detail: { values } }));
		} catch (err) {
			error = (err as Error).message;
		} finally {
			submitting = false;
		}
	}
</script>

{#if loading}
	<div class="animate-pulse h-20 bg-gray-100 rounded-lg"></div>
{:else}
	<form onsubmit={handleSubmit} class="space-y-4">
		{#each schema.fields as field (field.key)}
			<div>
				<label class="label" for="plugin-{field.key}">{label(field.label)}</label>

				{#if field.type === 'toggle'}
					<input
						id="plugin-{field.key}"
						type="checkbox"
						checked={!!values[field.key]}
						onchange={(e) => values[field.key] = (e.target as HTMLInputElement).checked}
						class="h-4 w-4 rounded border-gray-300 text-primary-600"
					/>
				{:else if field.type === 'select'}
					<select
						id="plugin-{field.key}"
						class="input"
						value={values[field.key] as string ?? ''}
						onchange={(e) => values[field.key] = (e.target as HTMLSelectElement).value}
						required={field.required}
					>
						<option value="">{label(field.placeholder) || '—'}</option>
						{#each field.options ?? [] as opt}
							<option value={opt.value}>{label(opt.label)}</option>
						{/each}
					</select>
				{:else if field.type === 'textarea'}
					<textarea
						id="plugin-{field.key}"
						class="input"
						placeholder={label(field.placeholder)}
						required={field.required}
						rows={3}
						oninput={(e) => values[field.key] = (e.target as HTMLTextAreaElement).value}
					>{values[field.key] as string ?? ''}</textarea>
				{:else}
					<input
						id="plugin-{field.key}"
						class="input"
						type={field.type === 'number' ? 'number' : field.type === 'password' ? 'password' : 'text'}
						placeholder={label(field.placeholder)}
						required={field.required}
						value={values[field.key] as string ?? ''}
						oninput={(e) => values[field.key] = (e.target as HTMLInputElement).value}
					/>
				{/if}

				{#if field.help_text}
					<p class="text-xs text-gray-500 mt-1">{label(field.help_text)}</p>
				{/if}
			</div>
		{/each}

		{#if error}
			<p class="text-red-600 text-sm">{error}</p>
		{/if}

		{#if schema.submit_url}
			<button type="submit" class="btn btn-primary" disabled={submitting}>
				{submitting ? '...' : 'Save'}
			</button>
		{/if}
	</form>
{/if}
