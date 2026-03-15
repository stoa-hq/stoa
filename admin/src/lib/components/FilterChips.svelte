<script lang="ts">
	import { X } from 'lucide-svelte';

	interface FilterOption {
		value: string;
		label: string;
	}

	interface Props {
		options: FilterOption[];
		selected: string;
		onSelect: (value: string) => void;
	}
	let { options, selected, onSelect }: Props = $props();
</script>

<div class="flex flex-wrap gap-2">
	{#each options as option}
		<button
			type="button"
			class="inline-flex items-center gap-1 rounded-full px-3 py-1 text-xs font-medium transition-all duration-150
				{selected === option.value
					? 'bg-primary-600 text-white shadow-sm'
					: 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-400 dark:hover:bg-gray-700'}"
			onclick={() => onSelect(option.value)}
		>
			{option.label}
			{#if selected === option.value && option.value !== ''}
				<X class="w-3 h-3" />
			{/if}
		</button>
	{/each}
</div>
