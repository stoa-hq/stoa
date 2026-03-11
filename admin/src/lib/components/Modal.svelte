<script lang="ts">
	import { fade, scale } from 'svelte/transition';

	interface Props {
		open: boolean;
		title: string;
		onClose: () => void;
		children?: import('svelte').Snippet;
		footer?: import('svelte').Snippet;
	}
	let { open, title, onClose, children, footer }: Props = $props();
</script>

{#if open}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="fixed inset-0 z-40 flex items-center justify-center p-4"
		transition:fade={{ duration: 150 }}
	>
		<div class="absolute inset-0 bg-black/40" onclick={onClose} onkeydown={(e) => e.key === 'Escape' && onClose()} role="button" tabindex="-1" aria-label="Modal schließen"></div>
		<div
			class="relative z-50 bg-white rounded-xl shadow-xl w-full max-w-lg"
			transition:scale={{ duration: 150, start: 0.95 }}
		>
			<div class="flex items-center justify-between px-6 py-4 border-b">
				<h2 class="text-lg font-semibold">{title}</h2>
				<button onclick={onClose} class="text-gray-400 hover:text-gray-600 text-xl leading-none">×</button>
			</div>
			<div class="px-6 py-4">
				{@render children?.()}
			</div>
			{#if footer}
				<div class="px-6 py-4 border-t flex justify-end gap-2">
					{@render footer()}
				</div>
			{/if}
		</div>
	</div>
{/if}
