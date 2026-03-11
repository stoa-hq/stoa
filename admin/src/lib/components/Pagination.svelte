<script lang="ts">
	import { t } from 'svelte-i18n';

	interface Props {
		currentPage: number;
		totalPages: number;
		onPageChange: (page: number) => void;
	}
	let { currentPage, totalPages, onPageChange }: Props = $props();
</script>

{#if totalPages > 1}
<div class="flex items-center justify-end px-1 py-3">
	<div class="flex gap-1">
		<button
			class="btn btn-secondary btn-sm"
			disabled={currentPage <= 1}
			onclick={() => onPageChange(currentPage - 1)}
		>‹ {$t('pagination.previous')}</button>
		{#each Array.from({ length: Math.min(totalPages, 7) }, (_, i) => {
			if (totalPages <= 7) return i + 1;
			if (currentPage <= 4) return i + 1;
			if (currentPage >= totalPages - 3) return totalPages - 6 + i;
			return currentPage - 3 + i;
		}) as p}
			<button
				class="btn btn-sm min-w-[2rem]"
				class:btn-primary={p === currentPage}
				class:btn-secondary={p !== currentPage}
				onclick={() => onPageChange(p)}
			>{p}</button>
		{/each}
		<button
			class="btn btn-secondary btn-sm"
			disabled={currentPage >= totalPages}
			onclick={() => onPageChange(currentPage + 1)}
		>{$t('pagination.next')} ›</button>
	</div>
</div>
{/if}
