<script lang="ts">
	interface Props {
		currentPage: number;
		totalPages: number;
		onPageChange: (page: number) => void;
	}
	let { currentPage, totalPages, onPageChange }: Props = $props();

	const pages = $derived(() => {
		const arr: (number | null)[] = [];
		for (let i = 1; i <= totalPages; i++) {
			if (i === 1 || i === totalPages || Math.abs(i - currentPage) <= 1) {
				arr.push(i);
			} else if (arr[arr.length - 1] !== null) {
				arr.push(null);
			}
		}
		return arr;
	});
</script>

<nav class="flex items-center justify-center gap-1">
	<button
		onclick={() => onPageChange(currentPage - 1)}
		disabled={currentPage <= 1}
		class="btn btn-secondary px-3 py-2 disabled:opacity-40"
	>
		‹
	</button>

	{#each pages() as p}
		{#if p === null}
			<span class="px-2 text-gray-400">…</span>
		{:else}
			<button
				onclick={() => onPageChange(p)}
				class="w-9 h-9 rounded-lg text-sm font-medium transition-colors
					{p === currentPage ? 'bg-primary-600 text-white' : 'text-gray-600 hover:bg-gray-100'}"
			>
				{p}
			</button>
		{/if}
	{/each}

	<button
		onclick={() => onPageChange(currentPage + 1)}
		disabled={currentPage >= totalPages}
		class="btn btn-secondary px-3 py-2 disabled:opacity-40"
	>
		›
	</button>
</nav>
