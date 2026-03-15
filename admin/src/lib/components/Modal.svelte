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
		<div class="absolute inset-0 bg-black/60 backdrop-blur-sm" onclick={onClose} onkeydown={(e) => e.key === 'Escape' && onClose()} role="button" tabindex="-1" aria-label="Modal schließen"></div>
		<div
			class="relative z-50 bg-[var(--surface)] dark:bg-[#1A1A2E]/90 dark:backdrop-blur-xl rounded-xl shadow-xl border border-[var(--card-border)] dark:border-gray-700/30 w-full max-w-lg"
			transition:scale={{ duration: 150, start: 0.95 }}
		>
			<div class="flex items-center justify-between px-6 py-4 border-b border-[var(--card-border)]">
				<h2 class="text-lg font-semibold text-[var(--text)]">{title}</h2>
				<button onclick={onClose} class="text-[var(--text-muted)] hover:text-[var(--text)] text-xl leading-none transition-colors">×</button>
			</div>
			<div class="px-6 py-4">
				{@render children?.()}
			</div>
			{#if footer}
				<div class="px-6 py-4 border-t border-[var(--card-border)] flex justify-end gap-2">
					{@render footer()}
				</div>
			{/if}
		</div>
	</div>
{/if}
