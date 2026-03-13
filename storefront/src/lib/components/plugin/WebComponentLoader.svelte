<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import type { UIComponent } from '$lib/stores/plugins';
	import { createPluginClient } from '$lib/api/plugin-client';

	interface Props {
		component: UIComponent;
		context?: Record<string, unknown>;
		onEvent?: (e: CustomEvent) => void;
	}

	let { component, context = {}, onEvent }: Props = $props();

	let containerEl: HTMLDivElement;
	let shadowRoot: ShadowRoot;

	const client = createPluginClient();

	function handlePluginEvent(e: Event) {
		if (e instanceof CustomEvent) {
			onEvent?.(e);
		}
	}

	onMount(async () => {
		shadowRoot = containerEl.attachShadow({ mode: 'closed' });

		// Load optional stylesheet
		if (component.style_url) {
			const link = document.createElement('link');
			link.rel = 'stylesheet';
			link.href = component.style_url;
			shadowRoot.appendChild(link);
		}

		// Load script with SRI verification
		const script = document.createElement('script');
		script.src = component.script_url;
		if (component.integrity) {
			script.integrity = component.integrity;
			script.crossOrigin = 'anonymous';
		}

		await new Promise<void>((resolve, reject) => {
			script.onload = () => resolve();
			script.onerror = () => reject(new Error(`Failed to load plugin script: ${component.script_url}`));
			document.head.appendChild(script);
		});

		// Create web component instance
		const el = document.createElement(component.tag_name);
		(el as any).context = context;
		(el as any).apiClient = client;
		el.addEventListener('plugin-event', handlePluginEvent);
		shadowRoot.appendChild(el);
	});

	onDestroy(() => {
		if (shadowRoot) {
			const el = shadowRoot.querySelector(component.tag_name);
			if (el) {
				el.removeEventListener('plugin-event', handlePluginEvent);
			}
		}
	});
</script>

<div bind:this={containerEl}></div>
