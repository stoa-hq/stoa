<script lang="ts">
	import { pluginStore, type UIExtension } from '$lib/stores/plugins';
	import SchemaForm from '$lib/components/plugin/SchemaForm.svelte';
	import WebComponentLoader from '$lib/components/plugin/WebComponentLoader.svelte';

	interface Props {
		slot: string;
		context?: Record<string, unknown>;
		onEvent?: (e: CustomEvent) => void;
	}

	let { slot, context = {}, onEvent }: Props = $props();

	const extensions = $derived(
		($pluginStore.extensions ?? []).filter((ext: UIExtension) => ext.slot === slot)
	);
</script>

{#each extensions as ext (ext.id)}
	{#if ext.type === 'schema' && ext.schema}
		<SchemaForm schema={ext.schema} {context} {onEvent} />
	{:else if ext.type === 'component' && ext.component}
		<WebComponentLoader component={ext.component} {context} {onEvent} />
	{/if}
{/each}
